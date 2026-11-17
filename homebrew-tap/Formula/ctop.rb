# Homebrew formula for ctop - Top-like interface for container metrics
# https://github.com/maxeo/ctop
#
# After a new release, update:
#   1. version variable
#   2. SHA256 checksums from the release's sha256sums.txt

class Ctop < Formula
  desc "Top-like interface for container metrics"
  homepage "https://github.com/maxeo/ctop"
  version "0.8.12"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-darwin-arm64"
      sha256 "d7dd3d90a8f1ed34f862516cd2da66241111a53ef747f40c042248e0a7f1c5c4"

      def install
        bin.install "ctop-#{version}-darwin-arm64" => "ctop"
      end
    else
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-darwin-amd64"
      sha256 "6167e5d2aa763e9bb367b3989f6fbdec517f4613cc27397d78db48b8272b7f08"

      def install
        bin.install "ctop-#{version}-darwin-amd64" => "ctop"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-linux-arm64"
      sha256 "17f9f95bbd773c6f19ae557ae796d305de08ec5820ec78c36778e2cefd28bc00"

      def install
        bin.install "ctop-#{version}-linux-arm64" => "ctop"
      end
    else
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-linux-amd64"
      sha256 "ae8697b35ab93806627abad2156fd3f301f3e001f047ae165ac01ec791a558bf"

      def install
        bin.install "ctop-#{version}-linux-amd64" => "ctop"
      end
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/ctop -v")
  end
end
