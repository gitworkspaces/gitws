# gitws — never mix work/personal git again

[![Release](https://img.shields.io/github/release/gitworkspaces/gitws.svg)](https://github.com/gitworkspaces/gitws/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A production-ready Go CLI that prevents mixing personal/work Git identities. Core functionality is **local/offline** with no backend required.

## Install

```bash
brew tap gitworkspaces/homebrew-tap
brew install gitws
```

## Quickstart

```bash
# Initialize workspaces
gitws init work --email you@work.com --host github
gitws init personal --email you@me.com --host github

# Clone repositories
gitws clone work microsoft/vscode
gitws clone personal myorg/myrepo

# Check status
gitws status
```

## Features

- **🔑 Per-workspace SSH keys**: Automatic generation and management
- **🔗 SSH aliases**: Clean, predictable host aliases (`github-work`, `gitlab-personal`)
- **👤 Per-repo identity**: Automatic user.name/user.email configuration
- **✍️ Signing support**: SSH and GPG commit signing
- **🛡️ Guard hooks**: Prevent accidental identity mixing
- **🔍 Doctor mode**: Diagnose and fix configuration issues
- **🔄 Key rotation**: Secure key rotation with backups

## Safety & Privacy

- **No telemetry**: All operations are local
- **Idempotent**: Safe to run multiple times
- **Backups**: Automatic backups before changes
- **Bounded markers**: Clear separation of managed vs manual config

## Uninstall

```bash
brew uninstall gitws
brew untap gitworkspaces/homebrew-tap
```

## Documentation

- [Full documentation](https://gitws.dev)
- [GitHub repository](https://github.com/gitworkspaces/gitws)
- [Report issues](https://github.com/gitworkspaces/gitws/issues)

## License

MIT License - see [LICENSE](LICENSE) for details.
