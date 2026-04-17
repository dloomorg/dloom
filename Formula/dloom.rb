class Dloom < Formula
  desc "Dotfile manager and system bootstrapper"
  homepage "https://github.com/dloomorg/dloom"
  url "https://github.com/dloomorg/dloom/archive/refs/tags/v0.0.9.rc.2.tar.gz"
  sha256 "562c7c00e1fd43647b65a59bc8780bbd6587bd3f83fad5e0c08fdd64ec9b5243"
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
