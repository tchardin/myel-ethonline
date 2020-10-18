# Chainlink Dev

This project contains tooling for Chainlink local development. It builds upon existing tooling and depends upon their
conventions (e.g. Ganache running at default port 7545).

## Dependencies

This project has a number of dependencies, including but not limited to: Node.js, Ganache and Docker.

## Usage

Before using this project, start a Ganache workspace that includes `./truffle/truffle-config.js` from this project and serves on all interfaces (`0.0.0.0`).

Run `npm start` to migrate the Truffle contracts and start a Chainlink node that is connected to Ganache.
Once started you can access the node by logging on to `locahost:6688` and entering the credentials listed in `chainlink/api.pwd`.

Run `npm run truffle:init-setup` to fund the deployed Chainlink client contract with `LINK`, fund your Chainlink node
with ETH and give the address of your Chainlink node fulfillment permissions on the deployed oracle contract.

Use your web browser and the Chainlink node web UI (`localhost:6688`) to add the following job to your Chainlink node
```json
{
  "initiators": [
    {
      "type": "runlog",
      "params": {
        "address": <YOUR_ORACLE_CONTRACT_ADDRESS>
      }
    }
  ],
  "tasks": [
    {
      "type": "httppost",
      "confirmations": 0,
      "params": {
        "post": <YOUR_LOTUS_NODE_API>
      }
    },
    {
      "type": "jsonparse",
      "confirmations": null,
      "params": {
        "path": [
          "result",
          "State",
          "MinSettleHeight"
        ]
      }
    },
    {
      "type": "ethuint256"
    },
    {
      "type": "ethtx"
    }
  ]
}
```

Replace `<YOUR_ORACLE_CONTRACT_ADDRESS>` with your oracle contract address, enclosed in quotation marks.
You can find this value on the configuration page for your Chainlink node.
Similarly replace `<YOUR_LOTUS_NODE_API>` with your lotus node API endpoint.

The credentials to login to the Chainlink node are `user@example.com/password`.

Run `npm run remix:truffle` to start a `remixd` server that will enable you to interact with your Truffle contracts
via the Remix web IDE. Use the Remix interface to connect Remix to Ganache and `remixd`. Open the
`GanacheChainlinkClient` contract and edit the `import` statements to allow Remix to resolve the dependencies from
GitHub and compile your contract. You will probably need to change the compiler version that Remix is using in order
for compilation to succeed. Once the contract has compiled, get the address of the deployed `ChannelVerify`
contract from Ganache and use it to load the deployed contract in Remix. Use your Chainlink node's oracle address and
the ID of the job you created to invoke the `verifyChannel` function on the deployed `ChannelVerify`
contract. Click the `transact` button multiple times in order to consume all the `LINK` your contract was funded with
and force your transactions through the mock Ganache blockchain. Use the Chainlink node web UI to confirm that your job
runs completed and are no longer in a pending state. Once at least one run of your job has completed, use Remix to view
the value of the `currentPrice` state variable on the deployed `ChannelVerify` contract.

Run `npm run stop` to shutdown the Chainlink node.

## Testing out Loans

Run the `npm run truffle:get-loan` to test out loan functions. The script will prompt you for values for the  `ORACLE_CONTRACT_ADDRESS`, `JOB_ID`, `CHANNEL` variables; in which you should infill your chainlink node's oracle contract address, the job id associated with the routine which verifies payment channels with a Lotus node (see above) , and the Filecoin payment channel address you want to submit for verification.

If successful you should see ETH transfers from your VerifyChannel contract to the 'user' account (usually account 0) on Ganache. A good way to verify this is with MetaMask, which plays nicely with Ganache !
