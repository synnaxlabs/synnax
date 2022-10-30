import Transport from '../transport';
import Retriever from './retriever';
/** The core client class for executing queries against a Synnax cluster ontology */
export default class OntologyClient extends Retriever {
    constructor(transport: Transport);
}
