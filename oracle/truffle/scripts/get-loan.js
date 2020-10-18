const GanacheChainlinkClient = artifacts.require('ChannelVerify');

module.exports = async callback => {
  console.log('Requesting Loan ...');
  const ganacheClient = await GanacheChainlinkClient.deployed();
  await ganacheClient.verifyChannel(process.argv[4], process.argv[5], process.argv[6])
  console.log(`Loan succeeded!`);
  callback();
}
