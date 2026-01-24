class Hydra < Formula
  desc "Fault-tolerant supervisor and proxy for MCP servers"
  homepage "https://github.com/proxikal/hydra"
  version "1.0.0"
  license "MIT"

  if OS.mac? && Hardware::CPU.intel?
    url "https://github.com/proxikal/hydra/releases/download/v1.0.0/hydra-darwin-amd64.tar.gz"
    sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_DARWIN_AMD64"
  elsif OS.mac? && Hardware::CPU.arm?
    url "https://github.com/proxikal/hydra/releases/download/v1.0.0/hydra-darwin-arm64.tar.gz"
    sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_DARWIN_ARM64"
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/proxikal/hydra/releases/download/v1.0.0/hydra-linux-amd64.tar.gz"
    sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_LINUX_AMD64"
  elsif OS.linux? && Hardware::CPU.arm?
    url "https://github.com/proxikal/hydra/releases/download/v1.0.0/hydra-linux-arm64.tar.gz"
    sha256 "REPLACE_WITH_ACTUAL_SHA256_FOR_LINUX_ARM64"
  end

  def install
    if OS.mac? && Hardware::CPU.intel?
      bin.install "hydra-darwin-amd64" => "hydra"
    elsif OS.mac? && Hardware::CPU.arm?
      bin.install "hydra-darwin-arm64" => "hydra"
    elsif OS.linux? && Hardware::CPU.intel?
      bin.install "hydra-linux-amd64" => "hydra"
    elsif OS.linux? && Hardware::CPU.arm?
      bin.install "hydra-linux-arm64" => "hydra"
    end
  end

  test do
    system "#{bin}/hydra", "--version"
  end

  def caveats
    <<~EOS
      Hydra has been installed!

      To get started:
        1. Initialize Hydra for your AI client:
           hydra init --client claude

        2. Start a supervised server:
           hydra run --name my-server

        3. Check status:
           hydra status my-server

      Configuration: ~/.hydra/config.json
      Documentation: https://github.com/proxikal/hydra

      For more help, run: hydra --help
    EOS
  end
end
