# Homebrew formula for ctop - Top-like interface for container metrics
# https://github.com/maxeo/ctop
#
# After a new release, update:
#   1. version variable
#   2. SHA256 checksums from the release's sha256sums.txt

class Ctop < Formula
  desc "Top-like interface for container metrics"
  homepage "https://github.com/maxeo/ctop"
  version "0.8.11"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-darwin-arm64"
      sha256 "5ee210be12700b0990a1b015442aacb3f289cfdd9ac7ab5279655c24150b288f"

      def install
        bin.install "ctop-#{version}-darwin-arm64" => "ctop"
      end
    else
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-darwin-amd64"
      sha256 "5bdb55e71f992eb508e10a52f4a93b05528f9d939ab7bed4820d995ac0cc1efa"

      def install
        bin.install "ctop-#{version}-darwin-amd64" => "ctop"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-linux-arm64"
      sha256 "2c9735b97008ab17abdd1cfff5b42015fb86e03a6d8f776f3164314b2ff53ecc"

      def install
        bin.install "ctop-#{version}-linux-arm64" => "ctop"
      end
    else
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-linux-amd64"
      sha256 "bd94faf33d88e66eec35b0d94068e82edc743405373a8e17894c3f903c090f75"

      def install
        bin.install "ctop-#{version}-linux-amd64" => "ctop"
      end
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/ctop -v")
  end
end
