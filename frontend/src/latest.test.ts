// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

import {describe, expect, it, vi} from "vitest";
import {LatestCall, type Cancellable} from "./latest";

/** A call that settles only when the test says so, and records cancel(). */
function deferred<T>() {
    let resolve!: (v: T) => void;
    let reject!: (e: unknown) => void;
    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });
    const cancel = vi.fn();
    const call: Cancellable<T> = Object.assign(promise, {cancel});
    return {call, resolve, reject, cancel};
}

/** Lets pending promise handlers run. */
const settle = () => new Promise((r) => setTimeout(r, 0));

describe("LatestCall", () => {
    it("drops an older response that arrives after a newer one", async () => {
        const latest = new LatestCall<string>();
        const onResult = vi.fn();
        const onError = vi.fn();
        const older = deferred<string>();
        const newer = deferred<string>();

        latest.run(older.call, onResult, onError);
        latest.run(newer.call, onResult, onError);
        expect(older.cancel).toHaveBeenCalledOnce();

        newer.resolve("new");
        await settle();
        older.resolve("old");
        await settle();

        expect(onResult).toHaveBeenCalledTimes(1);
        expect(onResult).toHaveBeenCalledWith("new");
        expect(onError).not.toHaveBeenCalled();
    });

    it("drops an older response that arrives before the newer one", async () => {
        const latest = new LatestCall<string>();
        const onResult = vi.fn();
        const older = deferred<string>();
        const newer = deferred<string>();

        latest.run(older.call, onResult, vi.fn());
        latest.run(newer.call, onResult, vi.fn());
        older.resolve("old");
        await settle();
        expect(onResult).not.toHaveBeenCalled();

        newer.resolve("new");
        await settle();
        expect(onResult).toHaveBeenCalledExactlyOnceWith("new");
    });

    it("drops the rejection of a superseded call, as a cancelled call rejects", async () => {
        const latest = new LatestCall<string>();
        const onError = vi.fn();
        const older = deferred<string>();
        const newer = deferred<string>();

        latest.run(older.call, vi.fn(), onError);
        latest.run(newer.call, vi.fn(), onError);
        older.reject(new Error("cancelled"));
        await settle();

        expect(onError).not.toHaveBeenCalled();
    });

    it("reports the newest call's rejection", async () => {
        const latest = new LatestCall<string>();
        const onResult = vi.fn();
        const onError = vi.fn();
        const only = deferred<string>();

        latest.run(only.call, onResult, onError);
        only.reject("boom");
        await settle();

        expect(onError).toHaveBeenCalledExactlyOnceWith("boom");
        expect(onResult).not.toHaveBeenCalled();
    });

    it("cancel() cancels the call in flight and drops its result", async () => {
        const latest = new LatestCall<string>();
        const onResult = vi.fn();
        const only = deferred<string>();

        latest.run(only.call, onResult, vi.fn());
        latest.cancel();
        expect(only.cancel).toHaveBeenCalledOnce();

        only.resolve("late");
        await settle();
        expect(onResult).not.toHaveBeenCalled();
    });

    it("does not report an error thrown by its own result handler", async () => {
        const latest = new LatestCall<string>();
        const onError = vi.fn();
        const only = deferred<string>();

        latest.run(only.call, () => {
            throw new Error("render failed");
        }, onError);
        only.resolve("x");
        await settle();

        expect(onError).not.toHaveBeenCalled();
    });
});
