class Vocabulary < Formula
  desc "Terminal flashcard app for studying English vocabulary"
  homepage "https://github.com/HiroyukiGPU/vocabulary_CLI"
  url "https://github.com/HiroyukiGPU/vocabulary_CLI/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "191998fc89b14393aac009aa17c45160f70801365a970d6ee843b77cca8e6842"
  version "0.1.0"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"vocabulary"), "."
  end

  test do
    output = shell_output("#{bin}/vocabulary help")
    assert_match "使い方:", output
    assert_match "vocabulary add", output
  end
end
