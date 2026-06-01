#include <stdio.h>
#include <string.h>
#define UNIX
#define SIZEOF_VOID_P 8
#include "CSP_WinDef.h"
#include "CSP_WinCrypt.h"
#include "WinCryptEx.h"

int main(void){
    DWORD i, type, cb;
    char name[512];

    printf("=== CryptEnumProviderTypes (the call the plugin uses) ===\n");
    for(i=0;;i++){
        type=0; cb=sizeof(name);
        if(!CryptEnumProviderTypes(i,0,0,&type,name,&cb)){
            printf("  stop at index %lu, GetLastError=0x%08lx\n",(unsigned long)i,(unsigned long)GetLastError());
            break;
        }
        printf("  type=%lu  name=%s\n",(unsigned long)type,name);
    }

    printf("\n=== CryptEnumProviders ===\n");
    for(i=0;;i++){
        type=0; cb=sizeof(name);
        if(!CryptEnumProviders(i,0,0,&type,name,&cb)){
            printf("  stop at index %lu, GetLastError=0x%08lx\n",(unsigned long)i,(unsigned long)GetLastError());
            break;
        }
        printf("  type=%lu  provider=%s\n",(unsigned long)type,name);
    }

    printf("\n=== CryptGetDefaultProvider(type 80) ===\n");
    cb=sizeof(name);
    if(CryptGetDefaultProvider(80,0,CRYPT_MACHINE_DEFAULT,name,&cb))
        printf("  type80 default = %s\n",name);
    else
        printf("  type80 FAILED GetLastError=0x%08lx\n",(unsigned long)GetLastError());
    return 0;
}
