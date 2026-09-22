// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

/** A promise that can be cancelled, such as a Wails binding call. */
export type Cancellable<T> = Promise<T> & {cancel(): void};

/**
 * Keeps at most one call in flight and delivers only the newest call's
 * outcome. Starting a call cancels the one before it, and a call that has
 * been superseded or cancelled never reaches its handlers, so an older
 * response that arrives late can't overwrite a newer one.
 */
export class LatestCall<T> {
    private pending: Cancellable<T> | null = null;

    /** Cancels the call in flight, if any, and drops its outcome. */
    cancel(): void {
        this.pending?.cancel();
        this.pending = null;
    }

    /**
     * Cancels any call in flight and tracks call instead. onResult or onError
     * runs only if call is still the newest when it settles.
     */
    run(call: Cancellable<T>, onResult: (value: T) => void, onError: (err: unknown) => void): void {
        this.cancel();
        this.pending = call;
        call.then((value) => {
            if (this.pending !== call) {
                return;
            }
            this.pending = null;
            onResult(value);
        }).catch((err) => {
            if (this.pending !== call) {
                return;
            }
            this.pending = null;
            onError(err);
        });
    }
}
