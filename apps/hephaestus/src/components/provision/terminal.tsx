
'use client';

import React, { useEffect, useRef } from 'react';
import { cn } from '@/lib/utils';
import { ScrollArea } from '@radix-ui/react-scroll-area';

interface TerminalProps {
    logs: string[];
    className?: string;
}

export function Terminal({ logs, className }: TerminalProps) {
    const bottomRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [logs]);

    return (
        <div className={cn("bg-black font-mono text-sm border border-zinc-800 rounded-lg overflow-hidden flex flex-col shadow-2xl", className)}>
            <div className="bg-zinc-900/50 px-4 py-2 border-b border-zinc-800 flex items-center justify-between">
                <div className="flex space-x-2">
                    <div className="w-3 h-3 rounded-full bg-red-500/20 border border-red-500/50"></div>
                    <div className="w-3 h-3 rounded-full bg-yellow-500/20 border border-yellow-500/50"></div>
                    <div className="w-3 h-3 rounded-full bg-green-500/20 border border-green-500/50"></div>
                </div>
                <span className="text-xs text-zinc-500 uppercase tracking-widest font-semibold">Build Logs</span>
            </div>

            <div className="p-4 overflow-y-auto flex-1 space-y-1 font-mono text-[13px]">
                {logs.length === 0 && (
                    <div className="text-zinc-600 italic">Waiting for logs...</div>
                )}

                {logs.map((log, i) => {
                    const isError = log.includes('ERR:') || log.includes('ERROR:');
                    const isStep = log.includes('Step ');
                    const isSuccess = log.includes('completed') || log.includes('Success');

                    // Basic timestamp parsing if present [HH:MM:SS]
                    let content = log;
                    let timestamp = '';
                    if (log.startsWith('[')) {
                        const closingIndex = log.indexOf(']');
                        if (closingIndex !== -1) {
                            timestamp = log.substring(1, closingIndex);
                            content = log.substring(closingIndex + 1).trim();
                        }
                    }

                    return (
                        <div key={i} className="flex group animate-in fade-in slide-in-from-left-2 duration-200">
                            {timestamp && (
                                <span className="text-zinc-600 mr-3 w-16 shrink-0 select-none group-hover:text-zinc-500 transition-colors">
                                    {timestamp}
                                </span>
                            )}
                            <span className={cn(
                                "break-all",
                                isError ? "text-red-400" :
                                    isStep ? "text-blue-400 font-semibold" :
                                        isSuccess ? "text-green-400" :
                                            "text-zinc-300"
                            )}>
                                {content}
                            </span>
                        </div>
                    );
                })}
                <div ref={bottomRef} />
            </div>
        </div>
    );
}
