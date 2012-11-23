# To run:
#
# $ bundle install
# $ bundle exec guard

guard :restarter, :command => "go run pastedown.go" do
  watch(/\.*\.go$/)
end

guard :shell do
  watch(%r{^coffee/.*\.coffee$}) do |f|
    `coffee -c -o public/ #{f[0]}`
    "Recompiling coffeescript."
  end
  watch(%r{^sass/(.*)\.scss$}) do |f|
    `bundle exec sass -r ./vendor/bourbon/lib/bourbon.rb sass/#{f[1]}.scss public/#{f[1]}.css`
    "Recompiling sass."
  end
end
