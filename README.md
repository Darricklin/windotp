# WindOTP

WindOTP 是为下面这个固定场景设计的 macOS 命令行工具：

- WindTerm 通过 SSH 登录 JumpServer
- JumpServer 显示 `Please enter 6 digits.`
- Google Authenticator 使用标准 6 位 TOTP
- 支持 WindTerm 检测提示后自动填写，也支持按快捷键填写

Base32 密钥只存放在 macOS Keychain 中。配置文件仅保存 profile 名称，不保存密钥。

WindOTP 支持两种使用方式，可同时配置：

- 自动填写：WindTerm Trigger 调用 `windotp trigger NAME`
- 快捷键填写：Automator 快速操作调用 `windotp auto`

两种方式共用同一套 profile、tab 绑定和 Keychain 密钥。

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
windotp add jump2
windotp bind jump1 jump-tap1
windotp bind jump2 jump-tap2
windotp list
```

`bind` 的第二个参数使用 WindTerm 顶部 tab 显示的名称。以后每增加一台 JumpServer，都需要执行
一次 `add` 和一次 `bind`。快捷键模式无需修改 Automator；自动模式还需要为新 session 添加一个
Trigger。

也可以从密码管理器通过 stdin 导入 Base32 或 `otpauth://` URL：

```bash
password-command | windotp add --stdin jump1
```

WindOTP 故意不提供 `--secret VALUE`，因为命令行参数可能被 shell history 或其他进程读取。

## 使用前检查

1. 确认 WindOTP 的安装路径，后面的 WindTerm 和 Automator 配置会用到：

   ```bash
   command -v windotp
   ```

   Apple Silicon Homebrew 通常返回 `/opt/homebrew/bin/windotp`，Intel Mac 通常返回
   `/usr/local/bin/windotp`。以下示例使用 `/opt/homebrew/bin/windotp`，请按实际结果替换。

2. 确认 profile 和默认 profile：

   ```bash
   windotp list
   ```

3. 在终端中生成一次验证码，和 Google Authenticator 当前显示的验证码比较：

   ```bash
   windotp code jump1
   ```

4. 检查配置文件、Keychain 和 macOS Accessibility：

   ```bash
   windotp doctor
   ```

   如果 macOS 弹出 Automation 或 Accessibility 请求，请允许 Terminal 控制 System Events。
   也可以在“系统设置 -> 隐私与安全性 -> 辅助功能”中手动启用 Terminal。

`windotp trigger` 和 `windotp auto` 会向当前前台应用发送按键，不能直接在 Terminal 中手动运行；
下面两种入口都会在输入前校验 WindTerm 是否位于前台。

## 方式一：WindTerm 自动填写

这种方式由每个 WindTerm session 自己检测验证码提示，不需要按快捷键。每个 session 都要绑定
对应的 profile，并单独创建 Trigger。

### 配置 jump1

1. 确认 `jump1` 已绑定到该 session 的 tab 标签：

   ```bash
   windotp bind jump1 jump-tap1
   ```

2. 在 WindTerm 中打开 `jump-tap1` 对应 session 的设置，进入 `Trigger` 或 `Triggers` 页面。
3. 新建一个 Trigger，填写：

   | 字段 | 值 |
   | --- | --- |
   | Pattern | `Please enter 6 digits\.` |
   | Action | `Run App` |
   | App | `/opt/homebrew/bin/windotp` |
   | Arguments | `trigger jump1` |

4. 启用 Trigger，保存 session 设置。
5. 保持 `jump-tap1` tab 在前台，重新发起 JumpServer 登录。当终端显示
   `Please enter 6 digits.` 时，WindOTP 会自动生成验证码、输入并按回车。
6. 第一次触发时，如果 macOS 请求 WindTerm 的 Automation 或 Accessibility 权限，请允许；如果
   没有弹窗，可在“系统设置 -> 隐私与安全性 -> 辅助功能”中启用 WindTerm，然后重新登录测试。

如果执行后提示 `detected labels` 中只有 `bash - WindTerm`，说明当前 WindTerm 版本没有向 macOS
暴露 tab 标签。把 Arguments 改为下面的配置即可跳过 tab 标签校验：

```text
Arguments: trigger --trust-profile jump1
```

`--trust-profile` 仍会确认 WindTerm 位于前台，但无法确认当前是哪一个 tab。只有在该 session 发起
登录时始终位于前台的情况下才能使用；后台 session 的 Trigger 可能把验证码输入到当前前台 tab。

### 配置 jump2

`jump2` 的配置相同，只需更换 tab 绑定和 Trigger 参数：

```bash
windotp bind jump2 jump-tap2
```

在 `jump-tap2` session 的 Trigger 中使用：

```text
Arguments: trigger jump2
```

如果 `jump2` 也无法读取 tab 标签，则使用 `trigger --trust-profile jump2`。

### 自动模式选项

- 默认等待 200ms，让 WindTerm 先处理完提示。终端较慢时可把 Arguments 改为
  `trigger --delay 500ms jump1`。
- 不希望自动按回车时，使用 `trigger --enter=false jump1`。
- Trigger 只处理它绑定的 profile。当前 tab 标签不匹配时会拒绝输入，避免把验证码发到错误 session。
- 后台 tab 触发时也会拒绝输入。切换到对应 tab 后，重新发起登录即可。
- `--trust-profile` 会关闭上面两项 tab 保护，只应作为 WindTerm 不暴露 tab 标签时的兼容模式。

不要让多个 Trigger 使用同一个 profile 参数。例如 `jump-tap2` session 不应配置
`trigger jump1`。

## 方式二：快捷键填写

使用 macOS 自带的 Automator。“快捷指令”的 Shell 操作运行在沙箱中，可能无法读取登录
Keychain，因此不适合作为 WindOTP 的入口。所有 profile 共用一个快速操作和一个快捷键，WindOTP
会根据当前选中 tab 的标签自动选择 profile。

1. 打开 Automator，新建“快速操作”。
2. 设置“工作流程收到当前：没有输入”“位于：任何应用程序”。
3. 添加“运行 Shell 脚本”，Shell 选择 `/bin/zsh`。
4. 填写下面的命令，把用户名和 `command -v windotp` 返回的安装路径替换为实际值：

   ```bash
   WINDOTP_CONFIG="/Users/你的用户名/Library/Application Support/windotp/config.json" /opt/homebrew/bin/windotp auto
   ```

5. 保存为 `WindOTP`。
6. 打开“系统设置 → 键盘 → 键盘快捷键 → 服务 → 通用”。
7. 勾选该服务，双击右侧空白处并按下快捷键，例如 `Control-Option-P`。
8. 回到 WindTerm，选中 `jump-tap1` 或 `jump-tap2` tab，发起登录。看到
   `Please enter 6 digits.` 后按快捷键，WindOTP 会自动识别 tab、输入验证码并按回车。
9. 第一次使用时，如果 macOS 请求 Automator、Automator Runner 或 System Events 权限，请允许；
   也可在“系统设置 -> 隐私与安全性 -> 辅助功能”中手动启用对应项目。

快捷键模式默认自动按回车。不希望自动按回车时，把 Automator 命令末尾改为
`windotp auto --enter=false`。无法唯一匹配当前 tab 时，WindOTP 会拒绝输入，不会弹出 profile
列表。

实际使用时，看到 `Please enter 6 digits.` 后按快捷键即可。在 M5 Mac、WindTerm 和多个 profile
的实际环境中，从按下快捷键到填写验证码约需 1 秒。

## 常见问题

- `WindTerm is not the frontmost application`：先点击目标 WindTerm tab，再重新触发登录或按快捷键。
- `no profile matches the active WindTerm tab`：运行 `windotp list` 检查 profile，再用
  `windotp bind NAME TAB_LABEL` 重新绑定；`TAB_LABEL` 必须出现在当前 tab 标签中。
- `detected labels` 只有 `bash - WindTerm` 等通用名称：WindTerm 没有向 Accessibility 暴露 tab
  标签。自动 Trigger 可使用 `trigger --trust-profile NAME`；必须确保触发登录的 session 位于前台。
- `cannot read the active WindTerm tab label` 或旧版本显示 `detected labels: []`：快捷键模式需要为
  Automator 授予 Accessibility 权限，Trigger 模式需要为 WindTerm 授权；授权后完全退出并重新打开
  Automator 和 WindTerm。
- `active WindTerm tab matches multiple profiles`：不同 profile 的绑定内容有重叠，改用更完整、唯一的
  tab 标签，例如 `jump-tap1` 和 `jump-tap2`。
- Accessibility 报错：在“系统设置 -> 隐私与安全性 -> 辅助功能”中允许实际调用 WindOTP 的
  Terminal、WindTerm 或 Automator，然后完全退出并重新打开对应应用。
- 验证码不正确：先运行 `windotp code NAME` 与 Google Authenticator 对比，并确认 Mac 的日期与时间
  设置为自动同步。

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
windotp trigger [--enter=true] [--min-validity=5s] [--delay=200ms] [--trust-profile] NAME
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
