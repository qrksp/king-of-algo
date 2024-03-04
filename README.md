## king of algo

### Dependencies

- python3 + pip to compile the pyteal code
- golang 1.22
- [gotask](https://taskfile.dev/)

### Environments

- mainnet: https://kingofalgo.com
- testnet: https://testnet.kingofalgo.com

### Integration test

Initiate by running the [algorand sandbox](https://github.com/algorand/sandbox).

Follow with:

```bash
$ task integration
```

### Motivation

As I am trying to improve my understanding of the Algorand blockchain, I took inspiration from the Ethereum-based game, [King of the Ether](https://www.kingoftheether.com). My project involves porting this game to the Algorand platform with a few rule adjustments. It's important to note that King of the Ether faced a security breach, detailed in their [postmortem analysis](https://www.kingoftheether.com/postmortem.html). Hopefully this game doesn't have the same fate.

### Bugs

If you encounter any bugs, I would greatly appreciate it if you could let me know.
