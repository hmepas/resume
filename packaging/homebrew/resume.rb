class Resume < Formula
  desc "Cross-agent AI coding session picker"
  homepage "https://github.com/hmepas/resume"
  version "0.1.7"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_arm64.tar.gz"
      sha256 "a9094abdb1e3c9a9f6464a6cd662291b3778bdc2d088f61c7033ac33ac3717a5"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_x86_64.tar.gz"
      sha256 "a929ae711d3e6cba0aa410ba4ed9fabf9d9b4c3bad9c2cc9850c43fb7f39fd81"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_arm64.tar.gz"
      sha256 "a9664ec3baa57bc6f809ac020a46b5e7c6df9cc41f084560174481432b80253a"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_x86_64.tar.gz"
      sha256 "710e18e5a2b6cc7fc2f19a0a089256a325e7307a79ac9e71126e12670ba2be1a"
    end
  end

  def install
    bin.install "resume"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/resume --version")
  end
end
