// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract Counter {
    uint256 public count;
    address public owner;

    event Incremented(address indexed by, uint256 newCount);

    constructor(uint256 start) {
        count = start;
        owner = msg.sender;
    }

    function increment() external {
        count += 1;
        emit Incremented(msg.sender, count);
    }

    function setCount(uint256 value) external {
        require(msg.sender == owner, "only owner");
        count = value;
    }
}
