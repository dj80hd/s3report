#!/usr/bin/env ruby
require 'optparse'
require 'aws-sdk'

class Options
  attr_reader :options

  def parse_options!(argv)
    parser = OptionParser.new
    parser.default_argv = argv

    parser.banner = 'Usage: s3report [options]'
    # TODO: Add more help on banner
    parser.on('-i', '--include substring', 'Include bucket names with this substring.  Default: include all') do |value|
      @options[:include] = value
    end
    parser.on('-e', '--exclude substring', 'Exclude bucket names with this substring.  Default: exclude none') do |value|
      @options[:exclude] = value
    end
    parser.on('-c', '--count n bstring', 'List the n newest (n positive) or n oldest (n negative) objects Default: 5 oldest') do |value|
      @options[:count] = value
    end
    parser.on('-j', '--json', 'json output') do |value|
      @options[:json] = true
    end

    parser.parse!
  end

  def count
    @options[:count]
  end

  def json
    return !@options[:json].nil?
  end

  def include
    @options[:include]
  end

  def exclude
    @options[:exclude]
  end

  def initialize(argv = ARGV)
    @options = { count: -5, include: '', exclude: '' }
    parse_options!(argv)
  end

end

# Monkey patch Integer to give human readable byte size (http://tinyurl.com/y7tg9ck6)
class Integer
  def to_filesize
    {
      'B'  => 1024,
      'KB' => 1024 * 1024,
      'MB' => 1024 * 1024 * 1024,
      'GB' => 1024 * 1024 * 1024 * 1024,
      'TB' => 1024 * 1024 * 1024 * 1024 * 1024
    }.each_pair { |e, s| return "#{(self.to_f / (s / 1024)).round(2)}#{e}" if self < s }
  end
end


class Bucket
  attr_reader :name, :creation_date

  def initialize(name, creation_date, objects = [])
    @name = name
    @creation_date = creation_date
    @last_modified = Time.new(1970, 1, 1)
    @objects = objects  
    @display_object_count = 0
    @size_per_owner_id = {}
    @total_size = 0
    @total_count = 0
    @error = nil
  end

  def analyze(n)
    begin
      @display_object_count = n
      @objects = objects.sort_by(&:last_modified)
      @last_modified = @objects[-1].last_modified unless @objects.empty?
      @objects.each do |o|

        @total_size += o.size
        @total_count += 1

        owner = o.dig(:owner, :id) || "unknown"
        @size_per_owner_id[owner] = 0 if @size_per_owner_id[owner].nil?
        @size_per_owner_id[owner] += o.size
      end

    rescue Exception => e
      @error = e.to_s + e.backtrace.to_s
    end

    # delete the objects we dont need anymore
    @objects = @display_object_count.to_i < 0 ? \
      @objects.first(@display_object_count.to_i.abs) : \
      @objects.last(@display_object_count.to_i.abs)

    self
  end

  def to_json
    {
      Name: @name,
      CreationDate: @creation_date,
      LastModified: @last_modified,
      TotalSize: @total_size,
      DispalyObjectCount: @display_object_count,
      TotalCount: @total_count,
      SizePerOwnerId: @size_per_owner_id,
      Objects: @objects.map { |o| o.last_modified.to_datetime.rfc3339 + ' ' + o.key },
      Error: @error
    }.to_json
  end 

  def to_s
    <<~HEREDOC
    Name: #{@name}
    CreationDate: #{@creation_date}
    LastModified: #{@last_modified}
    TotalSize: #{@total_size.to_filesize}
    TotalCount: #{@total_count}
    SizePerOwnerId:
    #{@size_per_owner_id.map{|k,v| " * " + k + " " + v.to_filesize}.join("\n")}
    Objects: 
    #{@objects.map{|o| " * #{o.last_modified.to_datetime.rfc3339} #{o.key}"}.join("\n")}
    Error: #{@error ? @error.to_s : "none"}
    HEREDOC
  end

  # Slurp all objects from s3 (api will only get 1000 at a time)
  def objects
    return @objects if ! @objects.empty?

    objects = []
    key_marker = nil
    begin
      more_objects = s3_service(name).list_objects(bucket: @name, marker: key_marker)

      if more_objects.contents
	objects += more_objects.contents.compact
      end

      key_marker = more_objects.contents[-1].key unless more_objects.contents.empty?

    end while more_objects.is_truncated 
    @objects = objects
  end
end

# Get a suitable s3 service for a bucket
# Buckets with a location constraint must use a service for that region
def s3_service(bucket_name = nil) 
  s3 = Aws::S3::Client.new
  if bucket_name
    resp = s3.get_bucket_location({bucket: bucket_name})
    region = resp['location_constraint']
    return Aws::S3::Client.new({region: region}) if !region.empty?
  end
  s3
end

# Get all buckets filtered by the --include and --exclude options
def get_all_buckets(include_substring, exclude_substring)
  s3_service.list_buckets['buckets'].select do |b|
    b.name.include?(include_substring) && (exclude_substring.empty? || !b.name.include?(exclude_substring))
  end.map { |b| Bucket.new(b.name, b.creation_date) }
end

def main
  options = Options.new

  buckets = get_all_buckets(options.include, options.exclude)

  # FIXME: Is this threadsafe ?  Will puts lock on stdout ?
  threads = buckets.map do |b|
    Thread.new {
      if options.json
        puts b.analyze(options.count).to_json
      else 
        puts b.analyze(options.count).to_s 
      end
    }
  end

  threads.each {|t| t.join}
end

main
