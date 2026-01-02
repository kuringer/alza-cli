# Security

## Reporting Vulnerabilities

If you find a security issue, please **do not** open a public issue.

Instead, use [GitHub's private vulnerability reporting](https://github.com/kuringer/alza-cli/security/advisories/new) or contact the maintainer directly.

## Token Storage

- Tokens are stored in `~/.config/alza/credentials.json`
- Keep this file private (permissions `600`)
- Never share or commit your tokens

## QuickBuy Safety

- Always use `--dry-run` first to verify
- Store payment config in `~/.config/alza/quickbuy.env`, never in code
