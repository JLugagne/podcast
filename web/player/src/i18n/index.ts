import { useSyncExternalStore } from "react";
import { readStorage, writeStorage } from "../lib/storage";

export type Locale = "en" | "fr";

const LOCALE_KEY = "locale";

/** Every user-facing string of the player, in English. */
const en = {
  appName: "Podcasts",
  skipToContent: "Skip to content",
  language: "Language",
  loading: "Loading…",
  somethingWentWrong: "Something went wrong",
  cancel: "Cancel",
  close: "Close",
  unexpectedError: "An unexpected error occurred.",
  tryAgain: "Try again",

  pageNotFound: "Page not found",
  pageNotFoundBody: "The page you requested does not exist.",
  backToLibrary: "Back to the library",

  listen: "Listen",
  librarySubtitle: "Your podcasts, saved on this device.",
  addPodcast: "Add podcast",
  addFirstPodcast: "Add your first podcast",
  loadingLibrary: "Loading your library…",
  couldNotLoadLibrary: "Could not load your library",
  emptyLibraryTitle: "Nothing to listen to yet",
  emptyLibraryBody: "Add a podcast to start listening.",
  continueListening: "Continue listening",
  seriesHeading: "Series",
  episodes: "{count} episodes",
  episodes_one: "{count} episode",
  episodes_other: "{count} episodes",
  unplayed: "{count} unplayed",
  unplayed_one: "{count} unplayed",
  unplayed_other: "{count} unplayed",
  resumeEpisode: "Resume {series} episode {number}: {title}",
  episodeKicker: "{series} · Episode {number}",
  chapterLine: "Chapter {number}: {title}",
  episodeLine: "Episode {number}",
  minutesLeft: "{count} min left",
  minutesLeft_one: "{count} min left",
  minutesLeft_other: "{count} min left",

  feedUrlLabel: "Podcast feed URL",
  feedUrlPlaceholder: "https://example.com/feed.xml",
  add: "Add",
  adding: "Adding…",
  couldNotAddPodcast: "Could not add this podcast",
  feedBlockedSummary: "Feed blocked by the browser?",
  corsProxyLabel: "CORS proxy prefix (optional)",
  corsProxyPlaceholder: "https://my-proxy.example.com/?url=",

  shareLibrary: "Share library",
  linkCopied: "Link copied",
  shareLinkLabel: "Share link",
  importingTitle: "Adding podcasts from this link…",
  feedsToCheck: "{count} feeds to check.",
  feedsToCheck_one: "{count} feed to check.",
  feedsToCheck_other: "{count} feeds to check.",
  importFailedTitle: "Could not add the podcasts from this link",
  importAdded: "{count} podcasts added",
  importAdded_one: "{count} podcast added",
  importAdded_other: "{count} podcasts added",
  importSkipped: "{count} podcasts already in your library",
  importSkipped_one: "{count} podcast already in your library",
  importSkipped_other: "{count} podcasts already in your library",
  importFailed: "{count} podcasts could not be added",
  importFailed_one: "{count} podcast could not be added",
  importFailed_other: "{count} podcasts could not be added",

  loadingSeries: "Loading series…",
  couldNotLoadSeries: "Could not load this series",
  seriesNotFound: "Series not found",
  seriesNotFoundBody: "This series does not exist or has been removed.",
  libraryLink: "Library",
  resume: "Resume",
  playLatest: "Play latest",
  resumeEpisodeLabel: "{action}: Episode {number}: {title}",
  refreshing: "Refreshing…",
  refresh: "Refresh",
  couldNotRefresh: "Could not refresh",
  remove: "Remove",
  removeConfirm: "Remove {title} from your library?",
  episodesHeading: "Episodes",
  episodeView: "Episode view",
  listView: "List",
  gridView: "Grid",
  noEpisodes: "No episodes",
  noPlayableEpisodes: "This feed has no playable episodes.",
  episodesOf: "Episodes of {title}",
  chapters: "{count} chapters",
  chapters_one: "{count} chapter",
  chapters_other: "{count} chapters",

  played: "Played",
  newEpisode: "New",
  pauseEpisode: "Pause episode {number}: {title}",
  playEpisode: "Play episode {number}: {title}",
  downloaded: "Downloaded",
  downloadingPercent: "Downloading {percent}%",
  downloading: "Downloading…",
  moreActions: "More actions for episode {number}",
  markPlayed: "Mark as played",
  markUnplayed: "Mark as unplayed",
  restartFromBeginning: "Restart from the beginning",
  removeDownload: "Remove download",
  downloadForOffline: "Download for offline",

  nowPlaying: "Now playing",
  closePlayer: "Close player",
  playbackProblem: "Playback problem",
  playerRegion: "Player",
  chapterOf: "Chapter {current} of {total}",
  noChapters: "No chapters",
  openChapterLink: "Open chapter link",
  previousChapter: "Previous chapter",
  back15: "Back 15 seconds",
  pause: "Pause",
  play: "Play",
  forward30: "Forward 30 seconds",
  nextChapter: "Next chapter",
  chaptersAndQueue: "Chapters and queue",
  chaptersHeading: "Chapters",
  upNext: "Up next",
  autoplayNext: "Autoplay next episode",
  playNext: "Play next: {title}",
  latestEpisode: "This is the latest episode of the series.",
  playbackSpeed: "Playback speed: {rate}",
  sleepTimer: "Sleep timer {time}",
  sleepTimerChapter: "Sleep timer end of chapter",
  sleepTimerEpisode: "Sleep timer end of episode",
  sleepTimerOff: "Sleep timer off",
  sleep: "Sleep",
  chapterWord: "Chapter",
  episodeWord: "Episode",
  sleepOff: "Off",
  minutes15: "15 minutes",
  minutes30: "30 minutes",
  minutes45: "45 minutes",
  minutes60: "60 minutes",
  endOfChapter: "End of chapter",
  endOfEpisode: "End of episode",

  miniPlayer: "Mini player",
  openPlayer: "Open player: {title}",
  noChaptersEpisode: "This episode has no chapters.",
  chapterAria: "Chapter {number}: {title}",
  playbackPosition: "Playback position",
  playbackPositionText: "{position} of {duration}, {chapter}",
  chapterText: "chapter {number}: {title}",
  noChapterText: "no chapter",

  invalidFeedUrl: "Enter a valid feed URL.",
  alreadyInLibrary: "This podcast is already in your library.",
  couldNotSaveLibrary: "Could not save the library (storage is full or unavailable).",
  notInLibrary: "This podcast is not in your library.",
  feedServerAnswered: "The feed server answered {status}.",
  feedDownloadFailed:
    "Could not download this feed. The server may block browser requests (CORS); set a CORS proxy in the settings.",
  invalidXmlFeed: "This URL is not a valid XML feed.",
  notPodcastFeed: "This XML document is not a podcast RSS feed.",
  untitledEpisode: "Untitled episode",
  offlineUnavailable: "Offline storage is not available here.",
  downloadCorsBlocked: "The audio server blocks downloads from the browser (CORS).",
  audioServerAnswered: "The audio server answered {status}.",
  notEnoughStorage: "Not enough storage space to keep this episode.",
  downloadFailed: "The download failed.",
  audioCouldNotPlay: "The audio could not be played.",
  noAudioFile: "This episode has no audio file.",
  nowPlayingChapter: "Now playing chapter {number}: {title}",
} as const;

export type MessageKey = keyof typeof en;
export type MessageParams = Record<string, string | number>;

const fr: Record<MessageKey, string> = {
  appName: "Podcasts",
  skipToContent: "Aller au contenu",
  language: "Langue",
  loading: "Chargement…",
  somethingWentWrong: "Une erreur est survenue",
  cancel: "Annuler",
  close: "Fermer",
  unexpectedError: "Une erreur inattendue est survenue.",
  tryAgain: "Réessayer",

  pageNotFound: "Page introuvable",
  pageNotFoundBody: "La page demandée n'existe pas.",
  backToLibrary: "Retour à la bibliothèque",

  listen: "Écouter",
  librarySubtitle: "Vos podcasts, enregistrés sur cet appareil.",
  addPodcast: "Ajouter un podcast",
  addFirstPodcast: "Ajouter votre premier podcast",
  loadingLibrary: "Chargement de votre bibliothèque…",
  couldNotLoadLibrary: "Impossible de charger votre bibliothèque",
  emptyLibraryTitle: "Rien à écouter pour le moment",
  emptyLibraryBody: "Ajoutez un podcast pour commencer à écouter.",
  continueListening: "Reprendre l'écoute",
  seriesHeading: "Séries",
  episodes: "{count} épisodes",
  episodes_one: "{count} épisode",
  episodes_other: "{count} épisodes",
  unplayed: "{count} non écoutés",
  unplayed_one: "{count} non écouté",
  unplayed_other: "{count} non écoutés",
  resumeEpisode: "Reprendre {series}, épisode {number} : {title}",
  episodeKicker: "{series} · Épisode {number}",
  chapterLine: "Chapitre {number} : {title}",
  episodeLine: "Épisode {number}",
  minutesLeft: "{count} min restantes",
  minutesLeft_one: "{count} min restante",
  minutesLeft_other: "{count} min restantes",

  feedUrlLabel: "URL du flux du podcast",
  feedUrlPlaceholder: "https://example.com/feed.xml",
  add: "Ajouter",
  adding: "Ajout…",
  couldNotAddPodcast: "Impossible d'ajouter ce podcast",
  feedBlockedSummary: "Flux bloqué par le navigateur ?",
  corsProxyLabel: "Préfixe du proxy CORS (facultatif)",
  corsProxyPlaceholder: "https://my-proxy.example.com/?url=",

  shareLibrary: "Partager la bibliothèque",
  linkCopied: "Lien copié",
  shareLinkLabel: "Lien de partage",
  importingTitle: "Ajout des podcasts de ce lien…",
  feedsToCheck: "{count} flux à vérifier.",
  feedsToCheck_one: "{count} flux à vérifier.",
  feedsToCheck_other: "{count} flux à vérifier.",
  importFailedTitle: "Impossible d'ajouter les podcasts de ce lien",
  importAdded: "{count} podcasts ajoutés",
  importAdded_one: "{count} podcast ajouté",
  importAdded_other: "{count} podcasts ajoutés",
  importSkipped: "{count} podcasts déjà dans votre bibliothèque",
  importSkipped_one: "{count} podcast déjà dans votre bibliothèque",
  importSkipped_other: "{count} podcasts déjà dans votre bibliothèque",
  importFailed: "{count} podcasts n'ont pas pu être ajoutés",
  importFailed_one: "{count} podcast n'a pas pu être ajouté",
  importFailed_other: "{count} podcasts n'ont pas pu être ajoutés",

  loadingSeries: "Chargement de la série…",
  couldNotLoadSeries: "Impossible de charger cette série",
  seriesNotFound: "Série introuvable",
  seriesNotFoundBody: "Cette série n'existe pas ou a été supprimée.",
  libraryLink: "Bibliothèque",
  resume: "Reprendre",
  playLatest: "Lire le plus récent",
  resumeEpisodeLabel: "{action} : épisode {number} : {title}",
  refreshing: "Actualisation…",
  refresh: "Actualiser",
  couldNotRefresh: "Impossible d'actualiser",
  remove: "Supprimer",
  removeConfirm: "Supprimer {title} de votre bibliothèque ?",
  episodesHeading: "Épisodes",
  episodeView: "Affichage des épisodes",
  listView: "Liste",
  gridView: "Grille",
  noEpisodes: "Aucun épisode",
  noPlayableEpisodes: "Ce flux ne contient aucun épisode lisible.",
  episodesOf: "Épisodes de {title}",
  chapters: "{count} chapitres",
  chapters_one: "{count} chapitre",
  chapters_other: "{count} chapitres",

  played: "Écouté",
  newEpisode: "Nouveau",
  pauseEpisode: "Mettre en pause l'épisode {number} : {title}",
  playEpisode: "Lire l'épisode {number} : {title}",
  downloaded: "Téléchargé",
  downloadingPercent: "Téléchargement {percent} %",
  downloading: "Téléchargement…",
  moreActions: "Plus d'actions pour l'épisode {number}",
  markPlayed: "Marquer comme écouté",
  markUnplayed: "Marquer comme non écouté",
  restartFromBeginning: "Reprendre depuis le début",
  removeDownload: "Supprimer le téléchargement",
  downloadForOffline: "Télécharger pour l'écoute hors ligne",

  nowPlaying: "Lecture en cours",
  closePlayer: "Fermer le lecteur",
  playbackProblem: "Problème de lecture",
  playerRegion: "Lecteur",
  chapterOf: "Chapitre {current} sur {total}",
  noChapters: "Aucun chapitre",
  openChapterLink: "Ouvrir le lien du chapitre",
  previousChapter: "Chapitre précédent",
  back15: "Reculer de 15 secondes",
  pause: "Pause",
  play: "Lire",
  forward30: "Avancer de 30 secondes",
  nextChapter: "Chapitre suivant",
  chaptersAndQueue: "Chapitres et file d'attente",
  chaptersHeading: "Chapitres",
  upNext: "À suivre",
  autoplayNext: "Lire automatiquement l'épisode suivant",
  playNext: "Lire ensuite : {title}",
  latestEpisode: "C'est le dernier épisode de la série.",
  playbackSpeed: "Vitesse de lecture : {rate}",
  sleepTimer: "Minuteur de sommeil {time}",
  sleepTimerChapter: "Minuteur de sommeil : fin du chapitre",
  sleepTimerEpisode: "Minuteur de sommeil : fin de l'épisode",
  sleepTimerOff: "Minuteur de sommeil désactivé",
  sleep: "Sommeil",
  chapterWord: "Chapitre",
  episodeWord: "Épisode",
  sleepOff: "Désactivé",
  minutes15: "15 minutes",
  minutes30: "30 minutes",
  minutes45: "45 minutes",
  minutes60: "60 minutes",
  endOfChapter: "Fin du chapitre",
  endOfEpisode: "Fin de l'épisode",

  miniPlayer: "Mini-lecteur",
  openPlayer: "Ouvrir le lecteur : {title}",
  noChaptersEpisode: "Cet épisode n'a pas de chapitres.",
  chapterAria: "Chapitre {number} : {title}",
  playbackPosition: "Position de lecture",
  playbackPositionText: "{position} sur {duration}, {chapter}",
  chapterText: "chapitre {number} : {title}",
  noChapterText: "aucun chapitre",

  invalidFeedUrl: "Saisissez une URL de flux valide.",
  alreadyInLibrary: "Ce podcast est déjà dans votre bibliothèque.",
  couldNotSaveLibrary: "Impossible d'enregistrer la bibliothèque (stockage plein ou indisponible).",
  notInLibrary: "Ce podcast n'est pas dans votre bibliothèque.",
  feedServerAnswered: "Le serveur du flux a répondu {status}.",
  feedDownloadFailed:
    "Impossible de télécharger ce flux. Le serveur bloque peut-être les requêtes du navigateur (CORS) ; configurez un proxy CORS dans les réglages.",
  invalidXmlFeed: "Cette URL n'est pas un flux XML valide.",
  notPodcastFeed: "Ce document XML n'est pas un flux RSS de podcast.",
  untitledEpisode: "Épisode sans titre",
  offlineUnavailable: "Le stockage hors ligne n'est pas disponible ici.",
  downloadCorsBlocked: "Le serveur audio bloque les téléchargements depuis le navigateur (CORS).",
  audioServerAnswered: "Le serveur audio a répondu {status}.",
  notEnoughStorage: "Espace de stockage insuffisant pour conserver cet épisode.",
  downloadFailed: "Le téléchargement a échoué.",
  audioCouldNotPlay: "L'audio n'a pas pu être lu.",
  noAudioFile: "Cet épisode n'a pas de fichier audio.",
  nowPlayingChapter: "Lecture du chapitre {number} : {title}",
};

const DICTIONARIES: Record<Locale, Record<MessageKey, string>> = { en, fr };

/** Picks the first supported language of a browser's preference list. */
export function localeFromLanguages(languages: readonly string[]): Locale {
  for (const language of languages) {
    const tag = language.toLowerCase();
    if (tag.startsWith("fr")) return "fr";
    if (tag.startsWith("en")) return "en";
  }
  return "en";
}

function detectLocale(): Locale {
  const stored = readStorage(LOCALE_KEY);
  if (stored === "en" || stored === "fr") return stored;
  if (typeof navigator === "undefined") return "en";
  const languages =
    navigator.languages && navigator.languages.length > 0
      ? navigator.languages
      : [navigator.language];
  return localeFromLanguages(languages);
}

let currentLocale: Locale = detectLocale();
const listeners = new Set<() => void>();

if (typeof document !== "undefined") document.documentElement.lang = currentLocale;

export function getLocale(): Locale {
  return currentLocale;
}

/** Changes the interface language and remembers the choice on this device. */
export function setLocale(locale: Locale): void {
  if (locale === currentLocale) return;
  currentLocale = locale;
  writeStorage(LOCALE_KEY, locale);
  if (typeof document !== "undefined") document.documentElement.lang = locale;
  for (const listener of listeners) listener();
}

export function subscribeLocale(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function pluralForm(count: number, locale: Locale): "one" | "other" {
  // French uses the singular for 0 as well as 1.
  if (locale === "fr") return Math.abs(count) <= 1 ? "one" : "other";
  return Math.abs(count) === 1 ? "one" : "other";
}

function interpolate(template: string, params?: MessageParams): string {
  if (!params) return template;
  return template.replace(/\{(\w+)\}/g, (placeholder, name: string) => {
    const value = params[name];
    return value === undefined ? placeholder : String(value);
  });
}

/** Translates a message in the current language, with `{name}` placeholders. */
export function t(key: MessageKey, params?: MessageParams): string {
  const dictionary = DICTIONARIES[currentLocale];
  const count = params?.count;
  if (typeof count === "number") {
    const pluralKey = `${key}_${pluralForm(count, currentLocale)}` as MessageKey;
    const pluralTemplate = dictionary[pluralKey];
    if (pluralTemplate !== undefined) return interpolate(pluralTemplate, params);
  }
  return interpolate(dictionary[key] ?? key, params);
}

/** Subscribes a component to language changes and returns the translator. */
export function useI18n(): { locale: Locale; t: typeof t; setLocale: typeof setLocale } {
  const locale = useSyncExternalStore(subscribeLocale, getLocale, getLocale);
  return { locale, t, setLocale };
}
