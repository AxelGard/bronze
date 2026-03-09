export function createGlobalEnv() {
    return {
        print: (...args: any[]) => {
            console.log(...args);
            return 0;
        },
    };
}
