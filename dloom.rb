class dloom < Formula
  desc "dloom - Weave your development environment."
  homepage "https://github.com/swaranga/dloom"
  url "https://github.com/swaranga/dloom/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "a7c8e70bff56d9139e7299ac482275171a1663c3e911f580b67df7f73bbd01d8"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", bin/"dloom"
  end

  test do
    system "#{bin}/dloom"
  end
end
