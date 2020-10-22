const GanacheChainlinkClient = artifacts.require('ChannelVerify');
const LinkToken = artifacts.require('LinkToken');

module.exports = async callback => {
  console.log(LinkToken.address);
  callback();
}
