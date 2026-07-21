class Windotp < Formula
  desc "Type JumpServer TOTP codes securely into WindTerm on macOS"
  homepage "https://github.com/Darricklin/windotp"
  head "https://github.com/Darricklin/windotp.git", branch: "main"

  depends_on :macos
  depends_on "go" => :build

  def install
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/windotp"
  end

  def caveats
    <<~EOS
      WindOTP needs macOS Automation/Accessibility permission to type into
      WindTerm. Run `windotp doctor` after adding your first profile.
    EOS
  end

  test do
    assert_match "windotp", shell_output("#{bin}/windotp version")
  end
end
