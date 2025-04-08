# Changelog
All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] Belukha - 2025-04-08 [:boom:]
This release introduces EVM channel support and other improvements.

### Added [:boom:]
- CI workflow [#10]
- Process potential events from transaction output [#12]
- Sender Interface to outsource transaction signing [#16]
- Multi-Asset support [#18]
- Allow one participant to withdraw for both parties [#22]
- Allow one participant to fund for both parties [#23]
 
### Changed [:boom:]
- Finalization of the Backend [#7]
- Update to go 1.21 [#8]
- Refactor of client package [#9]
- Adjust to go-perun's event and timeout types [#11]
- Remove global mutex from client package [#13]
- Separate signing and broadcasting of contract calls [#14]
- Update go go-perun v0.11.0 [#15]
- Adapt to cross-chain payment channels [#19]
- Remove channelID array [#21]
- Added support for EVM channels [#23]



[#7]: https://github.com/perun-network/perun-stellar-backend/pull/7
[#8]: https://github.com/perun-network/perun-stellar-backend/pull/8
[#9]: https://github.com/perun-network/perun-stellar-backend/pull/9
[#10]: https://github.com/perun-network/perun-stellar-backend/pull/10
[#11]: https://github.com/perun-network/perun-stellar-backend/pull/11
[#12]: https://github.com/perun-network/perun-stellar-backend/pull/12
[#13]: https://github.com/perun-network/perun-stellar-backend/pull/13
[#14]: https://github.com/perun-network/perun-stellar-backend/pull/14
[#15]: https://github.com/perun-network/perun-stellar-backend/pull/15
[#16]: https://github.com/perun-network/perun-stellar-backend/pull/16
[#18]: https://github.com/perun-network/perun-stellar-backend/pull/18
[#19]: https://github.com/perun-network/perun-stellar-backend/pull/19
[#21]: https://github.com/perun-network/perun-stellar-backend/pull/21
[#22]: https://github.com/perun-network/perun-stellar-backend/pull/22
[#23]: https://github.com/perun-network/perun-stellar-backend/pull/23

## [0.2.0] First Flight - 2024-02-16 [:boom:]
This release marks the completion of Stellar payment channels, using Soroban Smart Contracts to swap Stellar assets.

### Added
- Wallet Implementation by @janbormet in [#1]
- Wire Implementation (XDR encoding and Decoding of contract types) [#2]
- Channel backend [#3]
- Channel package finalization & Update to Horizon v2.27.0 [#6]

[#1]: https://github.com/perun-network/perun-stellar-backend/pull/1
[#2]: https://github.com/perun-network/perun-stellar-backend/pull/2
[#3]: https://github.com/perun-network/perun-stellar-backend/pull/3
[#6]: https://github.com/perun-network/perun-stellar-backend/pull/6


## Legend
- <span id="warning">:warning:</span> This is a pre-release and not intended for usage with real funds.
- <span id="breaking">:boom:</span> This is a breaking change, e.g., it changes the external API.
  [:warning:]: #warning
  [:boom:]: #breaking