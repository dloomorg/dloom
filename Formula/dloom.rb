class Dloom < Formula
  desc "Dotfile manager and system bootstrapper"
  homepage "https://github.com/dloomorg/dloom"
  url "https://github.com/dloomorg/dloom/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "b92b29ab163e197fe34f9af59ad54b057d58ecf515b7fdf37f5f3d86e9ec6191"
  license "MIT"
  head "https://github.com/dloomorg/dloom.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X github.com/dloomorg/dloom/cmd.Version=#{version}"), "-o", bin/"dloom"
  end

  test do
    assert_match "dloom version: #{version}", shell_output("#{bin}/dloom version")
  end
end
