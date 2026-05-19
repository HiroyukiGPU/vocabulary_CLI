class Vocabulary < Formula
  desc "Terminal flashcard app for studying English vocabulary"
  homepage "https://github.com/HiroyukiGPU/vocabulary_CLI"
  url "https://github.com/HiroyukiGPU/vocabulary_CLI.git",
      branch: "main",
      using: :git
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
