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
brew install go
make test
make install PREFIX="$(brew --prefix)"
```

Homebrew 6 不再接受仓库内的本地 Formula；项目建立独立 Homebrew tap 后再提供 `brew install`
方式。Apple Silicon 的 tag release 由 GitHub Actions 自动构建。

## 添加 JumpServer

交互输入不会回显，也不会进入 shell history：

```bash
windotp add --default jump1
windotp add staging
windotp bind jump1 jump-bj.sensetime.com
windotp bind staging jump-sh.sensetime.com
windotp list
```

也可以从密码管理器通过 stdin 导入 Base32 或 `otpauth://` URL：

```bash
password-command | windotp add --stdin jump1
```

WindOTP 故意不提供 `--secret VALUE`，因为命令行参数可能被 shell history 或其他进程读取。

## 手动验证

先在终端验证 TOTP 是否正确：

```bash
windotp code jump1
```

`windotp auto` 必须由下面的 Automator 快速操作调用。不要从 Terminal 手动调用，
因为此时 Terminal 而不是 WindTerm 位于前台。

首次输入时，macOS 可能要求 Automation 或 Accessibility 权限。按系统提示授权运行命令的
Terminal 或 Automator，然后执行：

```bash
windotp doctor
```

## 设置快捷键

使用 macOS 自带的 Automator。“快捷指令”的 Shell 操作运行在沙箱中，可能无法读取登录
Keychain，因此不适合作为 WindOTP 的入口。

1. 打开 Automator，新建“快速操作”。
2. 设置“工作流程收到当前：没有输入”“位于：任何应用程序”。
3. 添加“运行 Shell 脚本”。
4. 填写下面的命令，并将用户名和安装路径替换为实际值：

   ```bash
   WINDOTP_CONFIG="/Users/你的用户名/Library/Application Support/windotp/config.json" /usr/local/bin/windotp auto
   ```

5. 保存为 `WindOTP`。
6. 打开“系统设置 → 键盘 → 键盘快捷键 → 服务 → 通用”。
7. 勾选该服务，双击右侧空白处并按下快捷键，例如 `Control-Option-P`。

使用 `command -v windotp` 可确认实际安装路径。所有 JumpServer profile 共用这一个 Automator
快速操作和一个快捷键。WindOTP 根据 WindTerm 当前选中 tab 的标签自动匹配 profile，不弹出列表；
无法唯一匹配时会拒绝输入。

实际使用时，看到 `Please enter 6 digits.` 后按快捷键即可。

## WindTerm Trigger 自动填写（可选）

WindTerm 的 Trigger 支持匹配文本后运行外部程序。可创建 session 专属 Trigger：

- Pattern: `Please enter 6 digits\.`
- Action: Run App
- App: `/opt/homebrew/bin/windotp`
- Arguments: `type --delay 200ms jump1`

自动 Trigger 只适合当前登录 session 位于前台的情况。如果多个 tab 同时登录，后台 tab 的
Trigger 可能在前台 tab 输入验证码；WindOTP 只能确认前台应用是 WindTerm，无法可靠识别当前
WindTerm tab。因此多 session 环境推荐使用快捷键模式。

## 命令

```text
windotp add [--stdin] [--default] NAME
windotp list
windotp default NAME
windotp bind NAME WINDTERM_TAB_MATCH
windotp code [NAME]
windotp type [--enter=true] [--min-validity=5s] [--delay=0] [NAME]
windotp choose [--enter=true] [--min-validity=5s]
windotp auto [--enter=true] [--min-validity=5s]
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
