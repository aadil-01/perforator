import type { ToastProps } from '@gravity-ui/uikit';
import { toaster } from '@gravity-ui/uikit/toaster-singleton-react-18';


const HIDING_TIME = 10000;  // 10 seconds

export const createToast = (options: ToastProps) => toaster.add({
    autoHiding: HIDING_TIME,
    ...options,
});

export const createSuccessToast = (options: ToastProps) => createToast({
    theme: 'success',
    ...options,
});
