// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title Nuvex Token (NVX)
 * @notice Official ERC-20 Token of the Nuvex Blockchain
 * @dev Next Generation Layer-1 Blockchain
 *      Website: https://nuvex-chain.io
 *      Chain:   nuvex-1
 */

interface IERC20 {
    function totalSupply() external view returns (uint256);
    function balanceOf(address account) external view returns (uint256);
    function transfer(address to, uint256 amount) external returns (bool);
    function allowance(address owner, address spender) external view returns (uint256);
    function approve(address spender, uint256 amount) external returns (bool);
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
}

contract NuvexToken is IERC20 {
    // ── Token Info ──────────────────────────────
    string public constant name     = "Nuvex";
    string public constant symbol   = "NVX";
    uint8  public constant decimals = 18;

    // ── Supply ──────────────────────────────────
    // 500,000,000 NVX — Hard Cap wie auf der Nuvex Chain
    uint256 private constant MAX_SUPPLY = 500_000_000 * 10**18;

    // Initial: 55% Community = 275,000,000 NVX
    uint256 private constant INITIAL_SUPPLY = 275_000_000 * 10**18;

    // ── State ───────────────────────────────────
    uint256 private _totalSupply;
    mapping(address => uint256) private _balances;
    mapping(address => mapping(address => uint256)) private _allowances;

    // ── Burn ────────────────────────────────────
    uint256 public totalBurned;

    // ── Owner (anonym) ──────────────────────────
    address public owner;
    bool public mintingFinished = false;

    // ── Events ──────────────────────────────────
    event Burn(address indexed from, uint256 amount);
    event MintingFinished();

    // ── Constructor ─────────────────────────────
    constructor() {
        owner = msg.sender;
        // Community Allocation: 275,000,000 NVX sofort verfügbar
        _mint(msg.sender, INITIAL_SUPPLY);
    }

    // ── Modifiers ───────────────────────────────
    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    modifier canMint() {
        require(!mintingFinished, "Minting finished");
        require(_totalSupply < MAX_SUPPLY, "Max supply reached");
        _;
    }

    // ── ERC-20 Standard ─────────────────────────
    function totalSupply() external view override returns (uint256) {
        return _totalSupply;
    }

    function balanceOf(address account) external view override returns (uint256) {
        return _balances[account];
    }

    function transfer(address to, uint256 amount) external override returns (bool) {
        _transfer(msg.sender, to, amount);
        return true;
    }

    function allowance(address account, address spender) external view override returns (uint256) {
        return _allowances[account][spender];
    }

    function approve(address spender, uint256 amount) external override returns (bool) {
        _allowances[msg.sender][spender] = amount;
        emit Approval(msg.sender, spender, amount);
        return true;
    }

    function transferFrom(address from, address to, uint256 amount) external override returns (bool) {
        require(_allowances[from][msg.sender] >= amount, "Insufficient allowance");
        _allowances[from][msg.sender] -= amount;
        _transfer(from, to, amount);
        return true;
    }

    // ── Burn ────────────────────────────────────
    // 0.1% Burn auf jede Transaktion — wie auf Nuvex Chain
    function _transfer(address from, address to, uint256 amount) internal {
        require(from != address(0), "From zero address");
        require(to != address(0), "To zero address");
        require(_balances[from] >= amount, "Insufficient balance");

        // 0.1% Burn
        uint256 burnAmount = amount / 1000;
        uint256 transferAmount = amount - burnAmount;

        _balances[from] -= amount;
        _balances[to] += transferAmount;

        if (burnAmount > 0) {
            totalBurned += burnAmount;
            _totalSupply -= burnAmount;
            emit Burn(from, burnAmount);
            emit Transfer(from, address(0), burnAmount);
        }

        emit Transfer(from, to, transferAmount);
    }

    // ── Mint ────────────────────────────────────
    // Nur Owner kann minten (für Bridge, Staking Rewards etc.)
    function mint(address to, uint256 amount) external onlyOwner canMint {
        require(_totalSupply + amount <= MAX_SUPPLY, "Would exceed max supply");
        _mint(to, amount);
    }

    function _mint(address to, uint256 amount) internal {
        _totalSupply += amount;
        _balances[to] += amount;
        emit Transfer(address(0), to, amount);
    }

    // ── Finish Minting ──────────────────────────
    // Einmal aufgerufen — nie wieder neuer NVX mintbar
    function finishMinting() external onlyOwner {
        mintingFinished = true;
        emit MintingFinished();
    }

    // ── Transfer Ownership ──────────────────────
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Zero address");
        owner = newOwner;
    }

    // ── Renounce Ownership ──────────────────────
    // Macht den Contract vollständig dezentral
    function renounceOwnership() external onlyOwner {
        owner = address(0);
    }

    // ── Info ────────────────────────────────────
    function maxSupply() external pure returns (uint256) {
        return MAX_SUPPLY;
    }

    function circulatingSupply() external view returns (uint256) {
        return _totalSupply;
    }
}
