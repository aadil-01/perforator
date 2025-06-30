export interface Rum {
    finishDataLoading?: (value: string) => void;
    finishDataRendering?: (value: string) => void;
    makeSpaSubPage?: (value: string, options?: object, isBlock?: boolean, params?: Record<any, any>) => void;
    startDataRendering?: (value: string, renderType: string, shouldCall: boolean) => void;
    sendDelta?: (deltaName: string, value: number, params?: Record<string, any>) => void;
}

export const fakeRum: Rum = {};
