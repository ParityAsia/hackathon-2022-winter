// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/math/SafeMathUpgradeable.sol";

import "../interfaces/IPancakePair.sol";
import "../interfaces/IPancakeRouter02.sol";
import "../interfaces/IZap.sol";
import "../interfaces/IWETH.sol";


contract ZapArthswap is Initializable, OwnableUpgradeable, IZap {
    using SafeMathUpgradeable for uint;
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /* ========== CONSTANT VARIABLES ========== */
    //native assets
    address private constant wASTR = 0xAeaaf0e2c81Af264101B9129C00F4440cCF0F720;  //wASTR
    address private constant oUSD = 0x29F6e49c6E3397C3A84F715885F9F233A441165C;

    //cbridged assets
    address private constant USDC = 0x6a2d262D56735DbA19Dd70682B39F6bE9a931D98;
    address private constant BUSD = 0x4Bf769b05E832FCdc9053fFFBC78Ca889aCb5E1E;
    address private constant DAI = 0x6De33698e9e9b787e09d3Bd7771ef63557E148bb;
    
    IPancakeRouter02 private constant ROUTER = IPancakeRouter02(0xE915D2393a08a00c5A463053edD31bAe2199b9e7);

    /* ========== STATE VARIABLES ========== */

    mapping(address => bool) public notFlip;
    mapping(address => address) public routePairAddresses;
    address[] public tokens;

    bytes public lpToken1Payload; // token1()

    /* ========== INITIALIZER ========== */

    function initialize() public initializer{
        __Ownable_init_unchained();

        setRoutePairAddress(BUSD, USDC);
        setRoutePairAddress(DAI, USDC);
        setRoutePairAddress(oUSD, USDC);

        lpToken1Payload = abi.encodeWithSignature("token1()");
    }

    receive() external payable {}


    /* ========== View Functions ========== */

    // if is LP token
    function isFlip(address _address) public view returns (bool) {
        // return !notFlip[_address];
        (bool success, bytes memory data) = _address.staticcall(lpToken1Payload);
        return success && (abi.decode(data, (address)) != address(0));
    }

    function routePair(address _address) external view returns(address) {
        return routePairAddresses[_address];
    }

    /* ========== External Functions ========== */

    //ERC20 -> LP
    //ERC20 -> ERC20
    function zapInToken(address _from, uint amount, address _to) external override {
        IERC20Upgradeable(_from).safeTransferFrom(msg.sender, address(this), amount);
        _approveTokenIfNeeded(_from);

        if (isFlip(_to)) {
            IPancakePair pair = IPancakePair(_to);
            address token0 = pair.token0();
            address token1 = pair.token1();
            if (_from == token0 || _from == token1) {
                // swap half amount for other
                address other = _from == token0 ? token1 : token0;
                _approveTokenIfNeeded(other);
                uint sellAmount = amount.div(2);
                uint otherAmount = _swap(_from, sellAmount, other, address(this));
                pair.skim(address(this));
                ROUTER.addLiquidity(_from, other, amount.sub(sellAmount), otherAmount, 0, 0, msg.sender, block.timestamp);
            } else {
                uint ethAmount = _from == wASTR ? _unwrapETH(amount) : _swapTokenForETH(_from, amount, address(this));
                _swapETHToFlip(_to, ethAmount, msg.sender);
            }
        } else {
            _swap(_from, amount, _to, msg.sender);
        }
    }

    //ETH -> LP
    //ETH -> ERC20
    function zapIn(address _to) external payable override {
        _swapETHToFlip(_to, msg.value, msg.sender);
    }

    //LP -> ERC20
    //ERC20 -> ETH
    function zapOut(address _from, uint amount) external override {
        IERC20Upgradeable(_from).safeTransferFrom(msg.sender, address(this), amount);
        _approveTokenIfNeeded(_from);

        if (!isFlip(_from)) {
            _swapTokenForETH(_from, amount, msg.sender);
        } else {
            IPancakePair pair = IPancakePair(_from);
            address token0 = pair.token0();
            address token1 = pair.token1();

            if (pair.balanceOf(_from) > 0) {
                pair.burn(address(this));
            }

            if (token0 == wASTR || token1 == wASTR) {
                ROUTER.removeLiquidityETH(token0 != wASTR ? token0 : token1, amount, 0, 0, msg.sender, block.timestamp);
            } else {
                ROUTER.removeLiquidity(token0, token1, amount, 0, 0, msg.sender, block.timestamp);
            }
        }
    }

    /* ========== Private Functions ========== */

    function _approveTokenIfNeeded(address token) private {
        if (IERC20Upgradeable(token).allowance(address(this), address(ROUTER)) == 0) {
            IERC20Upgradeable(token).safeApprove(address(ROUTER), type(uint).max);
        }
    }

    function _swapETHToFlip(address flip, uint amount, address receiver) private {
        if (!isFlip(flip)) {
            _swapETHForToken(flip, amount, receiver);
        } else {
            // flip
            IPancakePair pair = IPancakePair(flip);
            address token0 = pair.token0();
            address token1 = pair.token1();
            if (token0 == wASTR || token1 == wASTR) {
                address token = token0 == wASTR ? token1 : token0;
                uint swapValue = amount.div(2);
                uint tokenAmount = _swapETHForToken(token, swapValue, address(this));

                _approveTokenIfNeeded(token);
                pair.skim(address(this));
                ROUTER.addLiquidityETH{value : amount.sub(swapValue)}(token, tokenAmount, 0, 0, receiver, block.timestamp);
            } else {
                uint swapValue = amount.div(2);
                uint token0Amount = _swapETHForToken(token0, swapValue, address(this));
                uint token1Amount = _swapETHForToken(token1, amount.sub(swapValue), address(this));

                _approveTokenIfNeeded(token0);
                _approveTokenIfNeeded(token1);
                pair.skim(address(this));
                ROUTER.addLiquidity(token0, token1, token0Amount, token1Amount, 0, 0, receiver, block.timestamp);
            }
        }
    }

    function _swapETHForToken(address token, uint value, address receiver) private returns (uint) {
        address[] memory path;

        if (routePairAddresses[token] != address(0)) {
            path = new address[](3);
            path[0] = wASTR;
            path[1] = routePairAddresses[token];
            path[2] = token;
        } else {
            path = new address[](2);
            path[0] = wASTR;
            path[1] = token;
        }

        uint[] memory amounts = ROUTER.swapExactETHForTokens{value : value}(1, path, receiver, block.timestamp);
        return amounts[amounts.length - 1];
    }

    function _swapTokenForETH(address token, uint amount, address receiver) private returns (uint) {
        address[] memory path;
        if (routePairAddresses[token] != address(0)) {
            path = new address[](3);
            path[0] = token;
            path[1] = routePairAddresses[token];
            path[2] = wASTR;
        } else {
            path = new address[](2);
            path[0] = token;
            path[1] = wASTR;
        }

        uint[] memory amounts = ROUTER.swapExactTokensForETH(amount, 1, path, receiver, block.timestamp);
        return amounts[amounts.length - 1];
    }

    function _swap(address _from, uint amount, address _to, address receiver) private returns (uint) {
        address intermediate = routePairAddresses[_from];
        if (intermediate == address(0)) {
            intermediate = routePairAddresses[_to];
        }

        address[] memory path;
        if (intermediate != address(0) && (_from == wASTR || _to == wASTR)) {
            path = new address[](3);
            path[0] = _from;
            path[1] = intermediate;
            path[2] = _to;
        } else if (intermediate != address(0) && (_from == intermediate || _to == intermediate)) {
            path = new address[](2);
            path[0] = _from;
            path[1] = _to;
        } else if (intermediate != address(0) && routePairAddresses[_from] == routePairAddresses[_to]) {
            path = new address[](3);
            path[0] = _from;
            path[1] = intermediate;
            path[2] = _to;
        } else if (routePairAddresses[_from] != address(0) && routePairAddresses[_to] != address(0) && routePairAddresses[_from] != routePairAddresses[_to]) {
            path = new address[](5);
            path[0] = _from;
            path[1] = routePairAddresses[_from];
            path[2] = wASTR;
            path[3] = routePairAddresses[_to];
            path[4] = _to;
        } else if (intermediate != address(0) && routePairAddresses[_from] != address(0)) {
            path = new address[](4);
            path[0] = _from;
            path[1] = intermediate;
            path[2] = wASTR;
            path[3] = _to;
        } else if (intermediate != address(0) && routePairAddresses[_to] != address(0)) {
            path = new address[](4);
            path[0] = _from;
            path[1] = wASTR;
            path[2] = intermediate;
            path[3] = _to;
        } else if (_from == wASTR || _to == wASTR) {
            path = new address[](2);
            path[0] = _from;
            path[1] = _to;
        } else {
            path = new address[](3);
            path[0] = _from;
            path[1] = wASTR;
            path[2] = _to;
        }

        uint[] memory amounts = ROUTER.swapExactTokensForTokens(amount, 1, path, receiver, block.timestamp);
        return amounts[amounts.length - 1];
    }

    function _unwrapETH(uint amount) private returns (uint) {
        require(IERC20Upgradeable(wASTR).balanceOf(address(this)) >= amount, "Zap: Not enough wASTR balance");
        uint beforeETH = address(this).balance;
        IWETH(wASTR).withdraw(amount);
        return (address(this).balance).sub(beforeETH);
    }

    /* ========== RESTRICTED FUNCTIONS ========== */

    function setRoutePairAddress(address asset, address route) public onlyOwner {
        routePairAddresses[asset] = route;
    }

    function setNotFlip(address token) public onlyOwner {
        bool needPush = notFlip[token] == false;
        notFlip[token] = true;
        if (needPush) {
            tokens.push(token);
        }
    }

    function removeToken(uint i) external onlyOwner {
        address token = tokens[i];
        notFlip[token] = false;
        tokens[i] = tokens[tokens.length - 1];
        tokens.pop();
    }

    function sweep() external onlyOwner {
        uint _length = tokens.length;
        for (uint i = 0; i < _length; i++) {
            address token = tokens[i];
            if (token == address(0)) continue;
            uint amount = IERC20Upgradeable(token).balanceOf(address(this));
            if (amount > 0) {
                _swapTokenForETH(token, amount, owner());
            }
        }
    }

    function withdraw(address token) external onlyOwner {
        if (token == address(0)) {
            payable(owner()).transfer(address(this).balance);
            return;
        }

        IERC20Upgradeable(token).transfer(owner(), IERC20Upgradeable(token).balanceOf(address(this)));
    }
}