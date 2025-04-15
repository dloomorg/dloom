class Dloom < Formula
  desc "Dotfile manager and system bootstrapper"
  homepage "https://github.com/dloomorg/dloom"
  url "https://github.com/dloomorg/dloom/archive/refs/tags/v0.0.8.tar.gz"
  sha256 "d7a6e79182a695bedb309e477ea8b8e2a30327b9803536c601d92428d38d7c2b"
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
