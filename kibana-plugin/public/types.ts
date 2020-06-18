import { NavigationPublicPluginStart } from '../../../src/plugins/navigation/public';

export interface ScorestackPluginSetup {
  getGreeting: () => string;
}
// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface ScorestackPluginStart {}

export interface AppPluginStartDependencies {
  navigation: NavigationPublicPluginStart;
}