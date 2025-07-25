# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Agent < Formula
  desc ""
  homepage ""
  version "0.0.6"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/jtarchie/agent/releases/download/v0.0.6/agent_Darwin_x86_64.tar.gz"
      sha256 "8afbcdf1f9820fddadda79b941d9891b7559ad73e6b14473b019cca301884792"

      def install
        bin.install "agent"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/jtarchie/agent/releases/download/v0.0.6/agent_Darwin_arm64.tar.gz"
      sha256 "99337646bf82a5c24ebe73d2ca74b183294e8ae2209a8535c0afcb6ea5fe2c66"

      def install
        bin.install "agent"
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel? and Hardware::CPU.is_64_bit?
      url "https://github.com/jtarchie/agent/releases/download/v0.0.6/agent_Linux_x86_64.tar.gz"
      sha256 "0c82acf2156df1c3c4d5ae7360fb89245b7b0c23348e7cbf01278d79075c307a"
      def install
        bin.install "agent"
      end
    end
    if Hardware::CPU.arm? and Hardware::CPU.is_64_bit?
      url "https://github.com/jtarchie/agent/releases/download/v0.0.6/agent_Linux_arm64.tar.gz"
      sha256 "0eea3be56a01667ac8ff4566164f4713edd08a5ea339e898ea4be901ff277da8"
      def install
        bin.install "agent"
      end
    end
  end

  test do
    system "#{bin}/agent --help"
  end
end
