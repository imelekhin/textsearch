# textsearch
Helper utility for find regexp and patterns in Kafka message (json). Do regexp (in golang format) search in specified fields of Kafka message or finds patterns loaded from file in specified field. If rule expression after "if" true when writes alarm message  to specified on start up Kafka topic. 
Notes.
1. Due to some optimisation in calculating logic expressions (calc only needed part, for example if one operand of "and" is false do not calculate second) and usage of reverse polish notation internally in code operand sequence and bracket distribution can make sugnificient impact on processing speed
2. Measured processing speed with 3 rule with simple regexps and one list with ~100000 paterns ~40-50 kEps on Intel(R) Xeon(R) CPU E5320  @ 1.86GHz     
3. In rule if statement  between operands, operators and brackets MUST BE SPACES
4. For lists loaded from files used aho corasick pattern match algorithm - no regexps!
5. Generally where is no full syntax error checks
6. cloudflare/ahokorasick has very high memory consumption. list of ~45 000 domains consume ~5 Gb memory

cfg file format:

\#declare regexp as variables to use in rules  
var   
 varname1 = regexp  
 varname2 = regexp  
endvar  

\# declare variables with regexps load from files  
list  
  varname3 = path_to_file  
  varname4 = path_to_file  
endlist  

\#rule declaration simple logic supported in if statement : & - AND, | - OR, = - equal, != - not equal  
rule rulename  
  if field1=varname1 & field2=varname2 | field3!=varname3  
  alarm Some text to put in alarm message  
endrule  



