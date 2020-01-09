# Pre Concept of this new blockchain before anycode is written

## The Full Node

For the purposes of this project, a full node has the job of maintaining a full copy of the blockchain, and all that entails as follows.
1. The node should receive and verify transactions to be added to the new block. 
2. The node should receive and verify all blocks from other nodes that have completed the proof of work. If the node has down time, on restart the node should request a copy of the blockchain from each node, verify all blocks in the chain, then pick the longest verified chain. In this way, all nodes have a fork of the blockchain. If the node finds an non-verifiable block, when looking for the longest chain, it should reply to the origin node with the header of the invalid block so the origin node can fix its copy of the chain.
3. The node should maintain and distribute a contact list of every full node that has submitted a block. This list will be used to distribute new blocks that the node as performed proof of work on. There is no guarantee that the nodes in the contact list are running honest code, or have an honest copy of the blockchain, but they have performed at least 1 proof of work for the block they submitted.
4. The block the node distributes must return with verification signatures from the other nodes for at least 70% of the contact list.