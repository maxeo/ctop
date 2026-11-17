# Homebrew formula for ctop - Top-like interface for container metrics
# https://github.com/maxeo/ctop
#
# After a new release, update:
#   1. version variable
#   2. SHA256 checksums from the release's sha256sums.txt

class Ctop < Formula
  desc "Top-like interface for container metrics"
  homepage "https://github.com/maxeo/ctop"
  version "0.8.14"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-darwin-arm64"
      sha256 "6776ed5c59761ea6f94f676db1bcab545d81ada61284c2d719072972a3a90a60"

      def install
        bin.install "ctop-#{version}-darwin-arm64" => "ctop"
      end
    else
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-darwin-amd64"
      sha256 "24631537e1da1798e07e673e036c7ce407b55a0ea90e132792c0eacd2a23c468"

      def install
        bin.install "ctop-#{version}-darwin-amd64" => "ctop"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-linux-arm64"
      sha256 "ec06a1c70143353f89be934d149337c7de3912f1c161fdc9d4a06f5f8e97e625"

      def install
        bin.install "ctop-#{version}-linux-arm64" => "ctop"
      end
    else
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-linux-amd64"
      sha256 "9271e1d4c2a61fb3e1a7f22a833305014ebdfa4dc0995f341f6d5b1e3c92e229"

      def install
        bin.install "ctop-#{version}-linux-amd64" => "ctop"
      end
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/ctop -v")
  end
end
