Gem::Specification.new do |spec|
  spec.name = "example-ruby-app"
  spec.version = "1.0.0"
  spec.authors = ["Example Author"]
  spec.summary = "Example Ruby application for testing stacktower"

  spec.files = Dir["lib/**/*"]
  spec.require_paths = ["lib"]

  spec.add_dependency "rails", "~> 7.1"
  spec.add_dependency "pg", "~> 1.1"
end

