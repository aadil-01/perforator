import React from 'react';

import { ToasterComponent, ToasterProvider } from '@gravity-ui/uikit';
import { toaster } from '@gravity-ui/uikit/toaster-singleton';

import { RouterProvider } from 'src/providers/RouterProvider/RouterProvider';
import { ThemeProvider } from 'src/providers/ThemeProvider/ThemeProvider';
import { UserSettingsProvider } from 'src/providers/UserSettingsProvider/UserSettingsProvider';

import type { PageProps } from '../Page/Page';

import './App.scss';


const AppImpl: React.FC<{}> = () => {
    const searchParams = new URLSearchParams(window.location.search);
    const embed = searchParams.get('embed') === '1';
    const pageProps: PageProps = {
        embed,
    };
    return (<RouterProvider pageProps={pageProps} />);
};

export const App: React.FC<{}> = () => {
    return (
        <UserSettingsProvider>
            <ThemeProvider>
                <ToasterProvider toaster={toaster}>
                    <AppImpl />
                    <ToasterComponent/>
                </ToasterProvider>
            </ThemeProvider>
        </UserSettingsProvider>
    );
};
