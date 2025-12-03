import { MeiliSearch } from 'meilisearch';

const MEILI_URL = process.env.NEXT_PUBLIC_MEILI_URL || '';
const MEILI_KEY = process.env.NEXT_PUBLIC_MEILI_KEY || '';

export const meiliClient = new MeiliSearch({
  host: MEILI_URL,
  apiKey: MEILI_KEY,
});

export const INDEX_ONTOLOGY = 'ontology';
export const INDEX_CLASSIFICATION = 'classification';
