# textsearch
Helper utility for TI when logs flows through Kafka. Do regexp (in golang format) search in specified fields of Kafka message. Messages must be in JSON format. If regexp true writes message  to specified Kafka topic. 
cfg file format:


[field] - field name in Kafka JSON message to search for regexps. Field names are case sensetive
 r:'regexp':comment - single regular expression regexp search in field. If found send message to specified Kfka topic with "comment"
 f:'filename":comment  - find all regular expressions listed in every line in file "filename". If found send message to specified Kfka topic with "comment"

