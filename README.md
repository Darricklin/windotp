# WindOTP

WindOTP 是为下面这个固定场景设计的 macOS 命令行工具：

- WindTerm 通过 SSH 登录 JumpServer
- JumpServer 显示 `Please enter 6 digits.`
- Google Authenticator 使用标准 6 位 TOTP
- 按一个快捷键生成、填写并提交验证码

Base32 密钥只存放在 macOS Keychain 中。配置文件仅保存 profile 名称，不保存密钥。

## 安装

当前源码安装方式：

```bash
git clone https://github.com/Darricklin/windotp.git
cd windotp
make test
make install PREFIX=/opt/homebrew
```

发布到 GitHub 后，可以通过项目内的 Homebrew Formula 安装 HEAD 版本：

```bash
brew install --HEAD ./Formula/windotp.rb
```

Apple Silicon 的 tag release 由 GitHub Actions 自动构建。正式发布稳定版 Homebrew Formula
时，需要加入对应 release tarball 的 URL 和 SHA256；也可以再创建独立的 Homebrew tap。

## 添加 JumpServer

交互输入不会回显，也不会进入 shell history：

```bash
windotp add --default production
windotp add staging
windotp list
```

也可以从密码管理器通过 stdin 导入 Base32 或 `otpauth://` URL：

```bash
password-command | windotp add --stdin production
```

WindOTP 故意不提供 `--secret VALUE`，因为命令行参数可能被 shell history 或其他进程读取。

## 手动验证

先在终端验证 TOTP 是否正确：

```bash
windotp code production
```

再打开 WindTerm，让光标停在 `Please enter 6 digits.` 后运行：

```bash
windotp type production
```

默认会输入六位码并按 Enter。如果验证码剩余有效期不足 5 秒，会等到下一个 30 秒周期。
WindTerm 不是前台应用时命令会拒绝输入。

首次输入时，macOS 可能要求 Automation 或 Accessibility 权限。按系统提示授权运行命令的
Terminal、Shortcuts 或 Automator，然后执行：

```bash
windotp doctor
```

## 设置快捷键

推荐用 macOS 自带的“快捷指令”：

1. 新建一个快捷指令，加入“运行 Shell 脚本”。
2. 脚本填写 `/opt/homebrew/bin/windotp type production`。
3. 在快捷指令详情中选择“添加键盘快捷键”，例如 `Control-Option-O`。
4. 为每个 JumpServer profile 建一个快捷指令，或者只给默认 profile 建一个。

实际使用时，看到 `Please enter 6 digits.` 后按快捷键即可。

## WindTerm Trigger 自动填写（可选）

WindTerm 的 Trigger 支持匹配文本后运行外部程序。可创建 session 专属 Trigger：

- Pattern: `Please enter 6 digits\.`
- Action: Run App
- App: `/opt/homebrew/bin/windotp`
- Arguments: `type --delay 200ms production`

自动 Trigger 只适合当前登录 session 位于前台的情况。如果多个 tab 同时登录，后台 tab 的
Trigger 可能在前台 tab 输入验证码；WindOTP 只能确认前台应用是 WindTerm，无法可靠识别当前
WindTerm tab。因此多 session 环境推荐使用快捷键模式。

## 命令

```text
windotp add [--stdin] [--default] NAME
windotp list
windotp default NAME
windotp code [NAME]
windotp type [--enter=true] [--min-validity=5s] [--delay=0] [NAME]
windotp remove NAME
windotp doctor
windotp version
```

配置文件位置为 `~/Library/Application Support/windotp/config.json`，权限为 `0600`。
Keychain service 名称为 `dev.windotp.totp`。

## 安全说明

- TOTP 密钥以 generic password 保存到登录 Keychain，并设置为仅设备解锁时可访问。
- 自动输入仅接受内部生成的恰好六位数字。
- AppleScript 通过 stdin 交给 `/usr/bin/osascript`，验证码不会出现在进程参数中。
- `code` 会把验证码打印到 stdout；只在调试或脚本确实需要时使用。
- 删除 profile 会同时删除对应 Keychain 项目。
