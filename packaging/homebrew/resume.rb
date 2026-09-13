class Resume < Formula
  desc "Cross-agent AI coding session picker"
  homepage "https://github.com/hmepas/resume"
  version "0.1.6"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_arm64.tar.gz"
      sha256 "501ab1149cbb512da8034f63080292b189a964aba48cc8ebfaa11eb533f6c446"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_x86_64.tar.gz"
      sha256 "7fd13732d6fa797588b1a58dc8a4e1b001abb414cb695cef5ed8354688a2cd9f"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_arm64.tar.gz"
      sha256 "5a2ff7d70f96c84d6c849d40d1c7cfeebc48a048484de5326f74a49c0047a174"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_x86_64.tar.gz"
      sha256 "71320bab7d8460cecdfdacda1d4a43e6f538cf5fdf6b71acd4e5f66f95e7c6af"
    end
  end

  def install
    bin.install "resume"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/resume --version")
  end
end
