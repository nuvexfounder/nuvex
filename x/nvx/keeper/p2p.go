package keeper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
//  P2P Netzwerk — Node Kommunikation
//
//  Wie es funktioniert:
//  1. Dein Node lauscht auf Port 26656
//  2. Andere Nodes verbinden sich
//  3. Sie tauschen Blöcke und TXs aus
//  4. Alle Nodes sind synchronisiert
// ─────────────────────────────────────────────

const (
	P2PPort        = "26656"
	MaxPeers       = 50
	PingInterval   = 30 * time.Second
	SyncInterval   = 5 * time.Second
)

// MessageType definiert die Art der P2P Nachricht
type MessageType string

const (
	MsgHandshake   MessageType = "handshake"
	MsgPing        MessageType = "ping"
	MsgPong        MessageType = "pong"
	MsgGetBlocks   MessageType = "get_blocks"
	MsgBlocks      MessageType = "blocks"
	MsgNewBlock    MessageType = "new_block"
	MsgNewTx       MessageType = "new_tx"
	MsgGetPeers    MessageType = "get_peers"
	MsgPeers       MessageType = "peers"
)

// P2PMessage ist eine Nachricht zwischen Nodes
type P2PMessage struct {
	Type      MessageType `json:"type"`
	ChainID   string      `json:"chain_id"`
	Height    int64       `json:"height"`
	Payload   []byte      `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	NodeID    string      `json:"node_id"`
}

// Peer repräsentiert einen verbundenen Node
type Peer struct {
	ID         string
	Address    string
	conn       net.Conn
	Height     int64
	Connected  time.Time
	LastSeen   time.Time
	IsOutbound bool
}

// P2PNode verwaltet alle P2P Verbindungen
type P2PNode struct {
	mu         sync.RWMutex
	nodeID     string
	address    string
	peers      map[string]*Peer
	blockchain *Blockchain
	mempool    *Mempool
	listener   net.Listener
	running    bool

	// Channels für Events
	newBlocks chan *Block
	newTxs    chan *PendingTx
}

// NewP2PNode erstellt einen neuen P2P Node
func NewP2PNode(nodeID string, blockchain *Blockchain, mempool *Mempool) *P2PNode {
	return &P2PNode{
		nodeID:     nodeID,
		address:    "0.0.0.0:" + P2PPort,
		peers:      make(map[string]*Peer),
		blockchain: blockchain,
		mempool:    mempool,
		newBlocks:  make(chan *Block, 100),
		newTxs:     make(chan *PendingTx, 1000),
	}
}

// ─────────────────────────────────────────────
//  Node starten
// ─────────────────────────────────────────────

// Start startet den P2P Node
func (n *P2PNode) Start() error {
	listener, err := net.Listen("tcp", n.address)
	if err != nil {
		return fmt.Errorf("P2P konnte nicht starten: %w", err)
	}

	n.listener = listener
	n.running = true

	fmt.Printf("[P2P] ✅ Node lauscht auf %s\n", n.address)
	fmt.Printf("[P2P] ✅ Node ID: %s\n", n.nodeID[:16]+"...")

	// Eingehende Verbindungen annehmen
	go n.acceptConnections()

	// Regelmässig Peers pingen
	go n.pingRoutine()

	// Chain synchronisieren
	go n.syncRoutine()

	return nil
}

// Stop stoppt den P2P Node
func (n *P2PNode) Stop() {
	n.running = false
	if n.listener != nil {
		n.listener.Close()
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, peer := range n.peers {
		peer.conn.Close()
	}
}

// ─────────────────────────────────────────────
//  Verbindungen annehmen
// ─────────────────────────────────────────────

func (n *P2PNode) acceptConnections() {
	for n.running {
		conn, err := n.listener.Accept()
		if err != nil {
			if n.running {
				fmt.Printf("[P2P] Verbindungsfehler: %v\n", err)
			}
			continue
		}

		go n.handleConnection(conn, false)
	}
}

func (n *P2PNode) handleConnection(conn net.Conn, isOutbound bool) {
	defer conn.Close()

	peer := &Peer{
		Address:    conn.RemoteAddr().String(),
		conn:       conn,
		Connected:  time.Now(),
		LastSeen:   time.Now(),
		IsOutbound: isOutbound,
	}

	// Handshake senden
	if err := n.sendHandshake(conn); err != nil {
		return
	}

	// Handshake empfangen
	msg, err := n.readMessage(conn)
	if err != nil || msg.Type != MsgHandshake {
		return
	}

	peer.ID = msg.NodeID
	peer.Height = msg.Height

	// Max Peers prüfen
	n.mu.Lock()
	if len(n.peers) >= MaxPeers {
		n.mu.Unlock()
		fmt.Printf("[P2P] Max Peers erreicht — Verbindung abgelehnt\n")
		return
	}
	n.peers[peer.ID] = peer
	n.mu.Unlock()

	fmt.Printf("[P2P] ✅ Peer verbunden: %s (Height: %d)\n",
		peer.Address, peer.Height)

	// Nachrichten verarbeiten
	n.handlePeer(peer)

	// Peer trennen
	n.mu.Lock()
	delete(n.peers, peer.ID)
	n.mu.Unlock()
	fmt.Printf("[P2P] Peer getrennt: %s\n", peer.Address)
}

// ─────────────────────────────────────────────
//  Mit Peer verbinden
// ─────────────────────────────────────────────

// Connect verbindet sich mit einem anderen Node
func (n *P2PNode) Connect(address string) error {
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Verbindung zu %s fehlgeschlagen: %w", address, err)
	}

	fmt.Printf("[P2P] Verbinde mit %s...\n", address)
	go n.handleConnection(conn, true)
	return nil
}

// ─────────────────────────────────────────────
//  Nachrichten verarbeiten
// ─────────────────────────────────────────────

func (n *P2PNode) handlePeer(peer *Peer) {
	for {
		msg, err := n.readMessage(peer.conn)
		if err != nil {
			return
		}

		peer.LastSeen = time.Now()

		switch msg.Type {
		case MsgPing:
			n.sendMessage(peer.conn, P2PMessage{
				Type:    MsgPong,
				ChainID: "nuvex-1",
				Height:  n.blockchain.Height(),
				NodeID:  n.nodeID,
			})

		case MsgPong:
			peer.Height = msg.Height

		case MsgGetBlocks:
			blocks := n.blockchain.GetRecentBlocks(20)
			data, _ := json.Marshal(blocks)
			n.sendMessage(peer.conn, P2PMessage{
				Type:    MsgBlocks,
				ChainID: "nuvex-1",
				Height:  n.blockchain.Height(),
				Payload: data,
				NodeID:  n.nodeID,
			})

		case MsgNewBlock:
			var block Block
			if err := json.Unmarshal(msg.Payload, &block); err == nil {
				fmt.Printf("[P2P] Neuer Block von %s: #%d\n",
					peer.Address, block.Height)
				n.newBlocks <- &block
			}

		case MsgNewTx:
			var tx PendingTx
			if err := json.Unmarshal(msg.Payload, &tx); err == nil {
				n.mempool.Submit(tx.From, tx.To, tx.Amount, tx.Fee, tx.Nonce)
			}

		case MsgGetPeers:
			n.sendPeerList(peer.conn)
		}
	}
}

// ─────────────────────────────────────────────
//  Broadcasting
// ─────────────────────────────────────────────

// BroadcastBlock sendet einen neuen Block an alle Peers
func (n *P2PNode) BroadcastBlock(block *Block) {
	data, err := json.Marshal(block)
	if err != nil {
		return
	}

	msg := P2PMessage{
		Type:    MsgNewBlock,
		ChainID: "nuvex-1",
		Height:  block.Height,
		Payload: data,
		NodeID:  n.nodeID,
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	sent := 0
	for _, peer := range n.peers {
		if err := n.sendMessage(peer.conn, msg); err == nil {
			sent++
		}
	}

	if sent > 0 {
		fmt.Printf("[P2P] Block #%d an %d Peers gesendet\n", block.Height, sent)
	}
}

// BroadcastTx sendet eine neue TX an alle Peers
func (n *P2PNode) BroadcastTx(tx *PendingTx) {
	data, _ := json.Marshal(tx)
	msg := P2PMessage{
		Type:    MsgNewTx,
		ChainID: "nuvex-1",
		Payload: data,
		NodeID:  n.nodeID,
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, peer := range n.peers {
		n.sendMessage(peer.conn, msg)
	}
}

// ─────────────────────────────────────────────
//  Routinen
// ─────────────────────────────────────────────

func (n *P2PNode) pingRoutine() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !n.running {
			return
		}

		n.mu.RLock()
		for _, peer := range n.peers {
			n.sendMessage(peer.conn, P2PMessage{
				Type:    MsgPing,
				ChainID: "nuvex-1",
				Height:  n.blockchain.Height(),
				NodeID:  n.nodeID,
			})
		}
		peerCount := len(n.peers)
		n.mu.RUnlock()

		if peerCount > 0 {
			fmt.Printf("[P2P] Ping gesendet an %d Peers\n", peerCount)
		}
	}
}

func (n *P2PNode) syncRoutine() {
	ticker := time.NewTicker(SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !n.running {
			return
		}

		n.mu.RLock()
		for _, peer := range n.peers {
			if peer.Height > n.blockchain.Height() {
				fmt.Printf("[P2P] Peer %s ist weiter (Height: %d vs %d) — sync...\n",
					peer.Address, peer.Height, n.blockchain.Height())
				n.sendMessage(peer.conn, P2PMessage{
					Type:    MsgGetBlocks,
					ChainID: "nuvex-1",
					Height:  n.blockchain.Height(),
					NodeID:  n.nodeID,
				})
			}
		}
		n.mu.RUnlock()
	}
}

// ─────────────────────────────────────────────
//  Hilfsfunktionen
// ─────────────────────────────────────────────

func (n *P2PNode) sendHandshake(conn net.Conn) error {
	return n.sendMessage(conn, P2PMessage{
		Type:      MsgHandshake,
		ChainID:   "nuvex-1",
		Height:    n.blockchain.Height(),
		NodeID:    n.nodeID,
		Timestamp: time.Now().UTC(),
	})
}

func (n *P2PNode) sendMessage(conn net.Conn, msg P2PMessage) error {
	msg.Timestamp = time.Now().UTC()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = conn.Write(data)
	return err
}

func (n *P2PNode) readMessage(conn net.Conn) (*P2PMessage, error) {
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return nil, fmt.Errorf("Verbindung getrennt")
	}
	var msg P2PMessage
	if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (n *P2PNode) sendPeerList(conn net.Conn) {
	n.mu.RLock()
	addresses := make([]string, 0, len(n.peers))
	for _, peer := range n.peers {
		addresses = append(addresses, peer.Address)
	}
	n.mu.RUnlock()

	data, _ := json.Marshal(addresses)
	n.sendMessage(conn, P2PMessage{
		Type:    MsgPeers,
		ChainID: "nuvex-1",
		Payload: data,
		NodeID:  n.nodeID,
	})
}

// ─────────────────────────────────────────────
//  Status
// ─────────────────────────────────────────────

type P2PStats struct {
	NodeID      string   `json:"node_id"`
	Address     string   `json:"address"`
	PeerCount   int      `json:"peer_count"`
	PeerList    []string `json:"peers"`
	ChainHeight int64    `json:"chain_height"`
	Running     bool     `json:"running"`
}

func (n *P2PNode) Stats() P2PStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]string, 0, len(n.peers))
	for _, peer := range n.peers {
		peers = append(peers, fmt.Sprintf("%s (height:%d)", peer.Address, peer.Height))
	}

	return P2PStats{
		NodeID:      n.nodeID,
		Address:     n.address,
		PeerCount:   len(n.peers),
		PeerList:    peers,
		ChainHeight: n.blockchain.Height(),
		Running:     n.running,
	}
}
