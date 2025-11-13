import i18next, { InitOptions } from 'i18next';
import resourcesToBackend from 'i18next-resources-to-backend';
import { languages, fallbackLng, defaultNS } from './settings';

let initialized = false;

/**
 * Initialize i18next for server-side rendering
 * @param locale - The locale to initialize with
 * @param namespaces - List of translation namespaces to load
 * @returns Initialized i18next instance
 */
export async function initI18nServer(locale: string, namespaces: string[] = [defaultNS]) {
  if (!languages.includes(locale as typeof languages[number])) {
    locale = fallbackLng;
  }

  if (!initialized) {
    i18next.use(
      resourcesToBackend((lng: string, ns: string) => 
        import(`../../../public/locales/${lng}/${ns}.json`)
      )
    );
    initialized = true;
  }

  const options: InitOptions = {
    lng: locale,
    fallbackLng,
    ns: namespaces,
    defaultNS,
    interpolation: { 
      escapeValue: false 
    },
    supportedLngs: languages,
  };

  if (!i18next.isInitialized) {
    await i18next.init(options);
  } else {
    await i18next.changeLanguage(locale);
  }

  return i18next;
}

export default i18next;
