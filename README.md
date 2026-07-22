# WindOTP

WindOTP 是为下面这个固定场景设计的 macOS 命令行工具：

- WindTerm 通过 SSH 登录 JumpServer
- JumpServer 显示 `Please enter 6 digits.` 或 `Please Enter MFA Code`
- Google Authenticator 使用标准 6 位 TOTP
- 支持 WindTerm 图形化 MFA 弹窗自动填写，也支持按快捷键填写

Base32 密钥只存放在 macOS Keychain 中。配置文件仅保存 profile 名称，不保存密钥。

WindOTP 支持两种使用方式，可同时配置：

- 图形化 MFA 弹窗：连接前事件调用 `windotp popup NAME`
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
windotp bind jump1 jump-server-1
windotp bind jump2 jump-server-2
windotp list
```

`bind` 的第二个参数使用 WindTerm 顶部 tab 显示的名称。以后每增加一台 JumpServer，都需要执行
一次 `add` 和一次 `bind`。快捷键模式无需修改 Automator；自动模式需要为新 session 添加一条
`Before connection` 运行命令规则。

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

   以下示例使用 `/usr/local/bin/windotp`。如果上面的命令返回其他路径，所有示例均应换成
   该完整路径。

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

`windotp popup` 和 `windotp auto` 会向当前前台应用发送按键，不能直接在 Terminal 中手动运行；
下面两种入口都会在输入前校验 WindTerm 是否位于前台。

## 方式一：WindTerm 自动填写

WindTerm 中央弹出带输入框的图形化登录窗口时，使用 `windotp popup`。每个 session 都要绑定
对应的 profile，并单独创建一条连接前规则。

`windotp popup` 通过 macOS Accessibility 等待 WindTerm 前台窗口出现包含
`Please enter 6 digits` 或 `Please Enter MFA Code` 的图形化弹窗和已聚焦的输入框，随后生成验证码、输入并按回车。它不会把
终端缓冲区中的相同文字误认为弹窗。填写动态码前，它会自动取消 MFA 弹窗中的
`Remember this step` / `记住这一步`，防止 WindTerm 保存会过期的一次性验证码。

以 `jump1` 为例，在 WindTerm 的触发器管理器中新建规则：

| 字段 | 值 |
| --- | --- |
| Type | `Before connection` / `连接前` |
| Action | `Run Command` / `运行命令` |
| macOS Command | `/usr/local/bin/windotp` |
| Arguments | `popup --trust-profile --timeout=90s --delay=0 --min-validity=0 jump1` |
| Session | 只选择 `jump1` 对应的 WindTerm session |

保存后完全关闭旧 session，再重新打开。连接前事件会启动一次等待任务；MFA 弹窗出现后自动填写。
安装路径必须以 `command -v windotp` 的输出为准。

如果旧版本已经让 WindTerm 保存了过期 MFA，升级后需要在 `SSH > 验证 > 已保存自动认证`
点击一次 `清除`，然后重新连接。账号和固定密码可以继续保存，只有 MFA 这一步必须保持未勾选；
后续双击 session 时，WindTerm 使用已保存的固定凭据，`windotp popup` 填写当前动态码。

如果 WindTerm 能可靠地向 Accessibility 暴露当前 tab 标签，可去掉 `--trust-profile`；否则必须保证
该 session 在弹窗出现前一直位于前台。等待超时默认是 60 秒，上例放宽为 90 秒。默认同时兼容
`Please enter 6 digits` 和 `Please Enter MFA Code`；其他弹窗提示可用 `--prompt TEXT`。为 `jump2`
配置时，只需把 Arguments 末尾的 `jump1` 改为 `jump2`，并只选择 `jump2` 对应的 session。

## 方式二：快捷键填写

使用 macOS 自带的 Automator。“快捷指令”的 Shell 操作运行在沙箱中，可能无法读取登录
Keychain，因此不适合作为 WindOTP 的入口。所有 profile 共用一个快速操作和一个快捷键，WindOTP
会根据当前选中 tab 的标签自动选择 profile。

1. 打开 Automator，新建“快速操作”。
2. 设置“工作流程收到当前：没有输入”“位于：任何应用程序”。
3. 添加“运行 Shell 脚本”，Shell 选择 `/bin/zsh`。
4. 填写下面的命令，把用户名和 `command -v windotp` 返回的安装路径替换为实际值：

   ```bash
   WINDOTP_CONFIG="/Users/你的用户名/Library/Application Support/windotp/config.json" /usr/local/bin/windotp auto
   ```

5. 保存为 `WindOTP`。
6. 打开“系统设置 → 键盘 → 键盘快捷键 → 服务 → 通用”。
7. 勾选该服务，双击右侧空白处并按下快捷键，例如 `Control-Option-P`。
8. 回到 WindTerm，选中 `jump-server-1` 或 `jump-server-2` tab，发起登录。看到
   `Please enter 6 digits.` 后按快捷键，WindOTP 会自动识别 tab、输入验证码并按回车。
9. 第一次使用时，如果 macOS 请求 Automator、Automator Runner 或 System Events 权限，请允许；
   也可在“系统设置 -> 隐私与安全性 -> 辅助功能”中手动启用对应项目。

快捷键模式默认自动按回车。不希望自动按回车时，把 Automator 命令末尾改为
`windotp auto --enter=false`。无法唯一匹配当前 tab 时，WindOTP 会拒绝输入，不会弹出 profile
列表。

实际使用时，看到 `Please enter 6 digits.` 后按快捷键即可。在 M5 Mac、WindTerm 和多个 profile
的实际环境中，从按下快捷键到填写验证码约需 1 秒。

## 常见问题

- 双击 session 后完全没有反应：确认规则类型为 `Before connection` / `连接前`，操作为
  `Run Command` / `运行命令`，规则已启用、已保存，且只绑定到对应 session。
- `popup` 超时：确认 `macOS Command` 与 `command -v windotp` 的输出一致，WindTerm 在前台，
  并已在“系统设置 -> 隐私与安全性 -> 辅助功能”中授权。登录耗时较长时增加
  `--timeout`；其他 MFA 文案使用 `--prompt TEXT`。
- MFA 中的“记住这一步”仍被勾选：运行 `windotp version` 确认已安装最新构建，然后完全关闭
  当前 session 再重新连接。旧进程不会自动替换为新版本。
- `WindTerm is not the frontmost application`：先点击目标 WindTerm tab，再重新发起登录或按快捷键。
- `no profile matches the active WindTerm tab`：运行 `windotp list` 检查 profile，再用
  `windotp bind NAME TAB_LABEL` 重新绑定；`TAB_LABEL` 必须出现在当前 tab 标签中。
- `cannot read the active WindTerm tab label`：快捷键模式需要为 Automator 授予 Accessibility 权限。自动弹窗
  模式可使用 `popup --trust-profile NAME`，但必须确保对应 session 始终位于前台。
- `active WindTerm tab matches multiple profiles`：不同 profile 的绑定内容有重叠，改用更完整、唯一的
  tab 标签，例如 `jump-server-1` 和 `jump-server-2`。
- Accessibility 报错：允许实际调用 WindOTP 的 Terminal、WindTerm 或 Automator，然后完全退出并重新打开
  对应应用。
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
windotp popup [--enter=true] [--min-validity=5s] [--timeout=60s] [--interval=200ms] [--delay=100ms] [--trust-profile] [--prompt=TEXT] NAME
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
