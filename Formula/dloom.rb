class Dloom < Formula
  desc "Dotfile manager and system bootstrapper"
  homepage "https://github.com/dloomorg/dloom"
  url "https://github.com/dloomorg/dloom/archive/refs/tags/v1.0.3.tar.gz"
  sha256 "75035d1f5eb1de02a8242fc7a259099be47ac8703a654a11c9b6ce4d3131c2e5"
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
