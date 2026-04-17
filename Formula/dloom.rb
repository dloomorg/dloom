class Dloom < Formula
  desc "Dotfile manager and system bootstrapper"
  homepage "https://github.com/dloomorg/dloom"
  url "https://github.com/dloomorg/dloom/archive/refs/tags/v0.0.9.rc.2.tar.gz"
  sha256 "d65b756717385ce8f9b71c0ecb342aa4daa35807eb99338d624388bb2e0c6517"
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
