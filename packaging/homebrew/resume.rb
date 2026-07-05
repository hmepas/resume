class Resume < Formula
  desc "Cross-agent AI coding session picker"
  homepage "https://github.com/hmepas/resume"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_arm64.tar.gz"
      sha256 "8ccff01bc2f2b22b68c5d3985f41244ca968bc83e0bf5ac35130339ee1cc814f"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_x86_64.tar.gz"
      sha256 "0a3d7a480b01657037f0099b81ab590a6a2515f0a9a9525855fa7ed6b112696d"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_arm64.tar.gz"
      sha256 "65baa5b4ea5eb68c48ce26d9b48640565bc68ec04e8600c32042fda0bcf49909"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_x86_64.tar.gz"
      sha256 "29632bae91a9cb2376a0f9c995b9639a3055c4bbe8ca91246ed9b9c6b31c4ade"
    end
  end

  def install
    bin.install "resume"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/resume --version")
  end
end
