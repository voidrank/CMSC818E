#!/usr/bin/env ruby

N = ARGV.shift.to_i

$i = 0
while $i < N do
  s = "go run replica.go -N #{N} -r #{$i} #{ARGV.join(" ")}&"
  puts s
  system(s)
  $i = $i+1
end
