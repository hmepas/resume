class Resume < Formula
  desc "Cross-agent AI coding session picker"
  homepage "https://github.com/hmepas/resume"
  version "0.1.8"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_arm64.tar.gz"
      sha256 "ca128ecc2ac045704dba889a0c5ef9ff44e62c97c0d81a759fd014bf7cd1d21a"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_x86_64.tar.gz"
      sha256 "c30d1e4376fd4adb70a50978eed3eba809c48be95c686524aa6a135b98f95df2"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_arm64.tar.gz"
      sha256 "f6b3ceb0bdc9095dd7bfc93a003467fa86cabeeb83d691ca496ea68edf086f59"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_x86_64.tar.gz"
      sha256 "1f4bd7105ee02148b2b4e647b71bdb6638067a5c961fd1252a5c84bcd8f3c62f"
    end
  end

  def install
    bin.install "resume"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/resume --version")
  end
end
