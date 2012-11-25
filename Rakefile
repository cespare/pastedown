task :default => ["build:all"]

def exec_with_message(message, command)
  puts message
  output = `#{command}`
  puts output unless output.empty?
end

FILES = %w[
  pastedown
  view.html
  vendor
  public
  files/about.markdown
  files/reference.markdown
]

OUTPUT = "pastedown_built.tgz"

namespace :build do
  task :server do
    exec_with_message("Building server...", "go build")
  end

  task :styles do
    exec_with_message(
      "Building stylesheets...",
      "bundle exec sass -r ./vendor/bourbon/lib/bourbon.rb sass/style.scss public/style.css"
    )
  end

  task :javascript do
    exec_with_message("Building javascript...", "coffee -c -o public/ coffee/*.coffee")
  end

  desc "Build Pastedown server into a tarball ready for copying to a server."
  task :all => [:server, :styles, :javascript] do
    exec_with_message("Tarring up files...", "tar czf #{OUTPUT} #{FILES.join(" ")}")
    puts "Done. Output is #{OUTPUT}"
  end
end
