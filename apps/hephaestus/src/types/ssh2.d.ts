declare module 'ssh2' {
    import { EventEmitter } from 'events';
    import { Readable, Writable } from 'stream';

    export interface ClientChannel extends EventEmitter {
        stdin: Writable;
        stdout: Readable;
        stderr: Readable;
        write(data: any): boolean;
        close(): void;
        on(event: string, listener: Function): this;
        on(event: 'data', listener: (data: Buffer) => void): this;
        on(event: 'close', listener: (code: number, signal: any) => void): this;
        on(event: 'exit', listener: (code: number, signal: any, coreDump: boolean, description: string) => void): this;
    }

    export interface ConnectConfig {
        host: string;
        port?: number;
        username?: string;
        privateKey?: string;
        readyTimeout?: number;
        [key: string]: any;
    }

    export class Client extends EventEmitter {
        connect(config: ConnectConfig): void;
        exec(command: string, callback: (err: Error | undefined, stream: ClientChannel) => void): boolean;
        end(): void;
        on(event: string, listener: Function): this;
        on(event: 'ready', listener: () => void): this;
        on(event: 'error', listener: (err: Error) => void): this;
    }
}
