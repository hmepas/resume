class Resume < Formula
  desc "Cross-agent AI coding session picker"
  homepage "https://github.com/hmepas/resume"
  version "0.1.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_arm64.tar.gz"
      sha256 "9840019f7443b1252c6a107ebda989119a383e0066c5fc919dd59133def258b0"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_x86_64.tar.gz"
      sha256 "4377749dcdfad5cabf78b8a1544ccce5f3ea74e837ff8caa4679b8be44f22682"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_arm64.tar.gz"
      sha256 "63296c7c7ab8984abe7b341a4d6f33f0c210cbd413fe9382de1c0c0bdbc680a1"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_x86_64.tar.gz"
      sha256 "4fcf0dfb9f98016e2dd0b920806e21c54372739900df5199fff53fcfbb5bfd92"
    end
  end

  def install
    bin.install "resume"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/resume --version")
  end
end
