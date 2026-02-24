# typed: false
# frozen_string_literal: true

class __FORMULA_CLASS__ < Formula
  desc "__PROJECT_DESC__"
  homepage "__PROJECT_HOMEPAGE__"
  version "__VERSION__"
  license "MIT"

  livecheck do
    url :stable
    strategy :github_latest
  end

  on_macos do
    if Hardware::CPU.intel?
      url "__BASE_URL__/__BINARY_NAME__-darwin-amd64.tar.gz"
      sha256 "__DARWIN_AMD64_SHA__"
    end
    if Hardware::CPU.arm?
      url "__BASE_URL__/__BINARY_NAME__-darwin-arm64.tar.gz"
      sha256 "__DARWIN_ARM64_SHA__"
    end
  end

  on_linux do
    if Hardware::CPU.intel? && Hardware::CPU.is_64_bit?
      url "__BASE_URL__/__BINARY_NAME__-linux-amd64.tar.gz"
      sha256 "__LINUX_AMD64_SHA__"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "__BASE_URL__/__BINARY_NAME__-linux-arm64.tar.gz"
      sha256 "__LINUX_ARM64_SHA__"
    end
  end

  def install
    bin.install "__BINARY_NAME__"
    generate_completions_from_executable(bin/"__BINARY_NAME__", "completion")
    prefix.install_metafiles
  end

  service do
    run [opt_bin/"__BINARY_NAME__", "__SERVICE_SUBCOMMAND__"]
    keep_alive true
    log_path var/"log/__BINARY_NAME__.log"
    error_log_path var/"log/__BINARY_NAME__.error.log"
    working_dir Dir.home
    environment_variables PATH: std_service_path_env
  end

  def caveats
    <<~EOS
      To start __BINARY_NAME__ as a background service:
        brew services start __FORMULA_NAME__

      Note: brew services runs under launchd/systemd and may not inherit your shell environment.
      For authentication, configure `api_key` explicitly in your config instead of relying on environment variables.

      To run __BINARY_NAME__ directly:
        __BINARY_NAME__ __SERVICE_SUBCOMMAND__
    EOS
  end

  test do
    system "#{bin}/__BINARY_NAME__", "version"
  end
end