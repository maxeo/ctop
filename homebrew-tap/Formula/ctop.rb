# Homebrew formula for ctop - Top-like interface for container metrics
# https://github.com/maxeo/ctop
#
# After a new release, update:
#   1. version variable
#   2. SHA256 checksums from the release's sha256sums.txt

class Ctop < Formula
  desc "Top-like interface for container metrics"
  homepage "https://github.com/maxeo/ctop"
  version "0.8.13"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-darwin-arm64"
      sha256 "e0f87de79c79c5d925871d92917326415a33935068d884f496ca805410d0e260"

      def install
        bin.install "ctop-#{version}-darwin-arm64" => "ctop"
      end
    else
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-darwin-amd64"
      sha256 "22c78b7e1cf3f3280c6ef0c0da537a3f5283f5d192dc81a906a68230a4c54652"

      def install
        bin.install "ctop-#{version}-darwin-amd64" => "ctop"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-linux-arm64"
      sha256 "70103baa9d65d6f66868bfdd1a0e3323ff61af337ae4064659bfb5fd7dc1eb0c"

      def install
        bin.install "ctop-#{version}-linux-arm64" => "ctop"
      end
    else
      url "https://github.com/maxeo/ctop/releases/download/v#{version}/ctop-#{version}-linux-amd64"
      sha256 "87cc4d92efbafadc522871fb14d8a6c97e2a4688296fae3abedf67178d0bb133"

      def install
        bin.install "ctop-#{version}-linux-amd64" => "ctop"
      end
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/ctop -v")
  end
end
