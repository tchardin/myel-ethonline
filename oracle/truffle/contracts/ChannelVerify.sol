pragma solidity 0.4.24;

// import 'https://github.com/smartcontractkit/chainlink/blob/develop/evm-contracts/src/v0.4/ChainlinkClient.sol';
// import 'https://github.com/smartcontractkit/chainlink/blob/develop/evm-contracts/src/v0.4/vendor/Ownable.sol';
import "chainlink/contracts/ChainlinkClient.sol";
import "chainlink/contracts/vendor/Ownable.sol";

contract ChannelVerify is ChainlinkClient, Ownable {
  uint256 constant private ORACLE_PAYMENT = 1 * LINK;

  uint256 public succeeded;

  event RequestVerifyChannelFulfilled(
    bytes32 indexed requestId,
    uint256 indexed settleHeight
  );

  constructor(address _link) public Ownable() {
    setChainlinkToken(_link);
  }

  function () public payable {
}

  function strConcat(string _a, string _b, string _c)
  internal returns (string)
  {
    bytes memory _ba = bytes(_a);
    bytes memory _bb = bytes(_b);
    bytes memory _bc = bytes(_c);
    string memory abcde = new string(_ba.length + _bb.length + _bc.length);
    bytes memory babcde = bytes(abcde);
    uint k = 0;
    for (uint i = 0; i < _ba.length; i++) babcde[k++] = _ba[i];
    for (i = 0; i < _bb.length; i++) babcde[k++] = _bb[i];
    for (i = 0; i < _bc.length; i++) babcde[k++] = _bc[i];
    return string(babcde);
}

  function verifyChannel(address _oracle, string _jobId, string _channel)
    public
    onlyOwner
  {
    Chainlink.Request memory req = buildChainlinkRequest(stringToBytes32(_jobId), this, this.fulfillVerifyChannel.selector);
    req.add("body",strConcat("{\"method\": \"Filecoin.StateReadState\", \"params\":[\"", _channel,"\", null], \"id\": 1 }" ));
    sendChainlinkRequestTo(_oracle, req, ORACLE_PAYMENT);
  }

  function fulfillVerifyChannel(bytes32 _requestId, uint256 _settleheight)
    public
    recordChainlinkFulfillment(_requestId)
  {
    emit RequestVerifyChannelFulfilled(_requestId, _settleheight);
    if (_settleheight == 0) {
      require(owner.call.value(1000000000000000000)(""), "Unable to transfer");
      succeeded = 1;
    }
    else {
      succeeded = 0;
    }
  }

  function getChainlinkToken() public view returns (address) {
    return chainlinkTokenAddress();
  }

  function withdrawLink() public onlyOwner {
    LinkTokenInterface link = LinkTokenInterface(chainlinkTokenAddress());
    require(link.transfer(msg.sender, link.balanceOf(address(this))), "Unable to transfer");
  }

  function cancelRequest(
    bytes32 _requestId,
    uint256 _payment,
    bytes4 _callbackFunctionId,
    uint256 _expiration
  )
    public
    onlyOwner
  {
    cancelChainlinkRequest(_requestId, _payment, _callbackFunctionId, _expiration);
  }

  function stringToBytes32(string memory source) private pure returns (bytes32 result) {
    bytes memory tempEmptyStringTest = bytes(source);
    if (tempEmptyStringTest.length == 0) {
      return 0x0;
    }

    assembly { // solhint-disable-line no-inline-assembly
      result := mload(add(source, 32))
    }
  }

}
