class Resume < Formula
  desc "Cross-agent AI coding session picker"
  homepage "https://github.com/hmepas/resume"
  version "0.1.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_arm64.tar.gz"
      sha256 "b8b4097a8f54374e7f5e4c089d6c97fb9703160740fc140f3268da6f729bbb6f"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_x86_64.tar.gz"
      sha256 "008baf9d32cdefbdceae7476110e178fc86279ca45da57b4b5383d3a21cf9d95"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_arm64.tar.gz"
      sha256 "9ff05a5224540ef37cd9defa51882a5ef235dbb92a053eac686df08f5fd9d0f2"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_x86_64.tar.gz"
      sha256 "23e6e0910cb4febd390c6a559b84425bf44fc0188fa63532533ea0e36305644e"
    end
  end

  def install
    bin.install "resume"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/resume --version")
  end
end
