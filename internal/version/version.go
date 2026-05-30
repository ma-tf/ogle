// Package version holds build metadata set at compile time via ldflags and
// the shared ASCII art branding.
package version

//nolint:gochecknoglobals // build metadata variables set via ldflags
var (
	// Version is the semantic version of the build.
	Version = "dev"
	// Commit is the git commit hash of the build.
	Commit = "none"
	// Date is the build date.
	Date = "unknown"
)

// ASCIIArt is the ogle brand ASCII art rendered in the About overlay and the
// CLI version command.
const ASCIIArt = `                         _               __         
       , ·. ,.-·~·.,   ‘              ,.-·^*ª'' ·,                 ,.  '                      _,.,  °    
      /  ·'´,.-·-.,   ','‚           .·´ ,·'´:¯''·,  '\‘            /   ';\               ,.·'´  ,. ,  ';\ '  
     /  .'´\:::::::'\   '\ °       ,´  ,'\:::::::::\,.·\'         ,'   ,'::'\            .´   ;´:::::\''´ \'\  
  ,·'  ,'::::\:;:-·-:';  ';\‚      /   /:::\;·'´¯''·;\:::\°      ,'    ;:::';'          /   ,'::\::::::\:::\:' 
 ;.   ';:::;´       ,'  ,':'\‚    ;   ;:::;'          '\;:·´      ';   ,':::;'          ;   ;:;:-·'~^ª*';\'´   
  ';   ;::;       ,'´ .'´\::';‚  ';   ;::/      ,·´¯';  °        ;  ,':::;' '          ;  ,.-·:*'´¨''*´\::\ '  
  ';   ':;:   ,.·´,.·´::::\;'°  ';   '·;'   ,.·´,    ;'\         ,'  ,'::;'            ;   ;\::::::::::::'\;'   
   \·,   '*´,.·'´::::::;·´     \'·.    ''´,.·:´';   ;::\'       ;  ';_:,.-·´';\‘     ;  ;'_\_:;:: -·^*';\   
    \\:¯::\:::::::;:·´         '\::\¯::::::::';   ;::'; ‘     ',   _,.-·'´:\:\‘    ';    ,  ,. -·:*'´:\:'\° 
     '\:::::\;::·'´  °            '·:\:::;:·´';.·´\::;'         \¨:::::::::::\';     \'*´ ¯\:::::::::::\;' '
         ¯                           ¯      \::::\;'‚          '\;::_;:-·'´‘         \:::::\;::-·^*'´     
          ‘                                    '\:·´'              '¨                    '*´¯              `
