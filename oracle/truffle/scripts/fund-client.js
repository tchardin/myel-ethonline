const GanacheChainlinkClient = artifacts.require('ChannelVerify');
const LinkToken = artifacts.require('LinkToken');

module.exports = async callback => {
  const ganacheClient = await GanacheChainlinkClient.deployed();
  const tokenAddress = await ganacheClient.getChainlinkToken();
  const token = await LinkToken.at(tokenAddress);
  console.log(`Transfering 5 LINK to ${ganacheClient.address}...`);
  const tx = await token.transfer(ganacheClient.address, `5000000000000000000`);
  console.log(`Transfer succeeded! Transaction ID: ${tx.tx}.`);

  const accounts = await web3.eth.getAccounts();
  console.log(`Sending 10 ETH from ${accounts[0]} to ${ganacheClient.address}.`);
  const result = await web3.eth.sendTransaction({from: accounts[0], to: ganacheClient.address, value: '1000000000000000000'});
  console.log(`Transfer succeeded! Transaction ID: ${result.transactionHash}.`);
  callback();
}
