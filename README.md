# textsearch
Helper utility for TI when logs flows through Kafka. Do regexp (in golang format) search in specified fields of Kafka message. Messages must be in JSON format. If rule expression after "if" true when writes alarm message  to specified Kafka topic. 
cfg file format:

# declare regexp as variables to use in rules
var  
 varname1 = regexp
 varname2 = regexp
endvar

# declare variables with regexps load from files
list
  varname3 = path_to_file
  varname4 = path_to_file
endlist

#rule declaration simple logic supported in if statement : & - AND, | - OR, = - equal, != - not equal
rule rulename
  if field1=varname1 & field2=varname2 | field3!=varname3
  alarm Some text to put in alarm message
endrule



