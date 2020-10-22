const GanacheChainlinkClient = artifacts.require("ChannelVerify");
let LinkToken = artifacts.require('LinkToken');

module.exports = function(deployer) {
  deployer.deploy(GanacheChainlinkClient, LinkToken.address);
};
