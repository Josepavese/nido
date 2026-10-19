# VM config reports updated but drops raw_qemu_args

Date: 2026-10-19

External-agent note from Half Pocket SDK qualification. No other file in this
repository was modified; this issue is untracked and has not been committed.

## Environment and evidence

Linux x86_64, installed `/home/jose/.nido/bin/nido` v4.5.27, protocol v3.0.
Read-only source inspection at b2a41f90d8b2fc2d7422631f95a2385daeb6c2e7.
VM `sdk-recovered-hub-20261019` belongs to this QA task.

MCP `nido_vm action=config_update raw_qemu_args=["-virtfs", "local,path=<private QA Data directory>,mount_tag=sdk_data,security_model=mapped-xattr,id=sdk_data"]`
reports `status=updated`. CLI gives the same result:

```sh
nido config sdk-recovered-hub-20261019 --qemu-arg=-virtfs --qemu-arg=local,path=/media/jose/Data/sdk-native-backup-store-20261019,mount_tag=sdk_data,security_model=mapped-xattr,id=sdk_data --json
nido info sdk-recovered-hub-20261019 --json
```

The config result is `updated` but `info.data.vm.raw_qemu_args` remains null.
Reproduced while running and stopped. After Nido stop/start the guest has no
9p virtio mount_tag; mounting sdk_data fails with `special device ... does not
exist`. Loading 9pnet_virtio does not reveal a tag.

Current source UpdateConfig in internal/provider/qemu.go updates many fields
and accelerators, but contains no assignment of updates.RawQemuArgs before
saveState writes state.RawQemuArgs. CLI/MCP both already populate the update.
This is a source observation, not a claim that the installed binary was built
from the current checkout.

## Required outcome

Honor an explicit raw-argument update, including explicit empty-list clearing;
keep an omitted field unchanged. Read-back and the following start must use
the saved value. Validate refusal/failure honestly rather than success with
unchanged state. Cover CLI and MCP paths, including stopped VM configuration.

## Decisions and impact

No upstream fix, build, install or update was attempted: only Half Pocket is
writable under the current mandate. No direct QEMU launch or hand-edited Nido
state workaround was used. The empty proposed host share was removed; no backup
was copied or moved. Existing VM data and backup locations remain unchanged.
The intent was bounded backup storage on Data to preserve root SSD space during
SDK acceptance. That storage branch remains unavailable through this method.
The Nido owner must decide implementation and distribution; an external agent
must not silently patch its installation or launch VMs outside Nido.

---

## Integrazione per il team Nido — 19 ottobre 2026, ore 11:51 CEST

**Nota di un agente esterno al progetto Nido.** Il proprietario ha chiesto di
consegnare questa analisi in `issues/`, affinche il team Nido prenda in carico
il difetto. Non ha autorizzato questo agente a correggere Nido. In questo
aggiornamento e' stato scritto soltanto questo file: nessuna modifica al codice,
ai test, alla configurazione, all'installazione o agli altri file di Nido;
nessuna build, commit, push, installazione o aggiornamento di Nido. Non sono
stati eseguiti comandi Git sul repository Nido. Il testo sopra conserva le
osservazioni della prima analisi; le verifiche nuove sono distinte sotto.

### Risultato atteso e risultato osservato

Il banco di accettazione SDK Half Pocket e' una VM posseduta da questa task.
Il caso d'uso richiede un percorso host su `/media/jose/Data` per i backup del
banco, senza avviare QEMU fuori Nido o modificare manualmente lo stato delle VM.
Si e' usato l'argomento QEMU `-virtfs` con una directory dedicata e il tag
`sdk_data`. Non e' una richiesta di accesso a directory aziendali o segreti.

L'aggiornamento tramite entrambe le superfici dichiarate, MCP e CLI, e' stato
accettato con esito `updated`. La successiva lettura CLI non conteneva gli
argomenti: `info.data.vm.raw_qemu_args` era `null`. Il comportamento e' stato
osservato sia con VM in esecuzione sia con VM ferma. Un successivo stop/start
gestito da Nido non ha reso disponibile il tag 9p nel guest. Il mount falliva
per dispositivo/tag inesistente; anche dopo il caricamento del modulo
`9pnet_virtio` il tag non compariva. Non si e' quindi trattato soltanto di una
configurazione salvata ma differita fino al riavvio.

La conclusione circoscritta e': la richiesta viene recepita dalla superficie
pubblica, ma il campo non arriva alla configurazione persistita/riusata per
l'avvio. Il messaggio di successo non dimostra che la modifica sia applicata.
Non si sostiene che ogni altra opzione di `config_update` abbia lo stesso
difetto, ne che 9p sia gia qualificato dopo la sola correzione del campo.

### Riproduzione disponibile per il manutentore

La prima analisi ha usato Nido installato v4.5.27, protocollo v3.0, Linux x86_64.
I comandi sotto sono una descrizione riproducibile, **non sono stati rieseguiti
da questo aggiornamento**. Non usare il banco custodito per nuove mutazioni
senza coordinarne dati e stato; e' preferibile una VM sacrificabile del team.

1. Predisporre una directory host dedicata, vuota e con permessi espliciti,
   su un filesystem con spazio sufficiente. Il precedente percorso proposto
   era `/media/jose/Data/sdk-native-backup-store-20261019`.
2. Su VM ferma inviare tramite MCP:

   ```json
   {
     "action": "config_update",
     "name": "sdk-recovered-hub-20261019",
     "raw_qemu_args": [
       "-virtfs",
       "local,path=/media/jose/Data/sdk-native-backup-store-20261019,mount_tag=sdk_data,security_model=mapped-xattr,id=sdk_data"
     ]
   }
   ```

3. Confrontare richiesta, risposta e read-back persistito. Provare separatamente
   il percorso CLI, gia riportato sopra, con i due `--qemu-arg=...`.
4. Riavviare soltanto attraverso Nido e verificare nel guest la presenza del
   tag prima del mount. Distinguere l'argomento salvato da quello effettivamente
   passato al processo QEMU e dal dispositivo realmente disponibile nel guest.
5. Eseguire anche omissione del campo e svuotamento esplicito, per evitare che
   la correzione introduca una cancellazione involontaria durante aggiornamenti
   di campi non correlati.

L'originaria directory host proposta, essendo vuota e propria della prova, e'
stata rimossa dopo il fallimento. Non contiene un backup da recuperare e la sua
eventuale assenza attuale non e' la causa dell'osservazione storica: al momento
del tentativo era stata predisposta. Non sono stati copiati o spostati backup
per aggirare il problema.

### Localizzazione della causa e limiti della diagnosi

L'ispezione precedente era riferita al sorgente alla revisione
`b2a41f90d8b2fc2d7422631f95a2385daeb6c2e7`. In
`internal/provider/qemu.go`, `UpdateConfig` aggiorna altri campi, compresi gli
acceleratori, ma non assegna `updates.RawQemuArgs` allo stato prima della
persistenza. `saveState` salva `state.RawQemuArgs`: in assenza dell'assegnazione,
il valore precedente rimane invariato. I percorsi CLI/MCP gia popolavano il
campo di aggiornamento. Il punto di convergenza del provider e' quindi il primo
confine da verificare; una correzione limitata alla CLI lascerebbe scoperto MCP.

Questa e' una localizzazione dal sorgente osservato, non una prova di provenienza
del binario installato. Non e' stato dimostrato che `/home/jose/.nido/bin/nido`
sia compilato esattamente da quella revisione. Il team deve verificare sorgente
corrente, build installata e riproduzione prima di dichiarare chiusa la causa.
Non sono stati riletti sorgenti o rieseguiti controlli Git di Nido in questo
aggiornamento; non si attribuisce al checkout odierno lo stato storico.

### Semantica da conservare nella correzione

- Campo omesso: mantenere gli argomenti correnti, senza cancellarli per effetto
  di un aggiornamento CPU/RAM o di altro campo indipendente.
- Lista non vuota: usare e persistere esattamente la lista approvata, preservando
  ordine e separazione degli argomenti; niente concatenazione tramite shell,
  duplicazione o append implicito ai valori precedenti.
- Lista vuota esplicita: consentire la rimozione intenzionale degli argomenti.
  Verificare che codec JSON, struct di update e rappresentazione CLI distinguano
  effettivamente omissione e svuotamento. Un campo con `omitempty` o una slice
  senza presenza esplicita puo perdere questa distinzione: e' un rischio da
  verificare, non una seconda causa gia riprodotta.
- Richiesta non supportata/non valida: rifiuto esplicito prima degli effetti.
  Non dichiarare `updated` se il provider scarta il campo.
- VM accesa: chiarire se la configurazione vale dal prossimo avvio. Non
  dichiarare hot-plug quando viene soltanto salvata una configurazione futura.
- Persistenza e concorrenza: conservare identita, dischi/backing, altre opzioni
  e garanzie di salvataggio del provider. Non introdurre un writer parallelo
  dei file di stato o una modifica manuale agli argomenti del processo attivo.

### Prove di accettazione suggerite al team

1. Test provider: lista presente, omissione, svuotamento e aggiornamento di un
   campo indipendente. Riapertura dello stato, non solo confronto in memoria.
2. Test superficie: CLI e MCP raggiungono la stessa semantica, senza adattatori
   che ignorino `[]` o trasformino argomenti in una stringa unica.
3. Prova su VM sacrificabile: update da ferma, read-back, avvio Nido, argomenti
   effettivi, tag 9p nel guest e lettura/scrittura di un file sintetico nella
   sola directory host dedicata. Poi chiusura, clear esplicito, nuovo avvio e
   assenza del tag, preservando il resto della configurazione.
4. Se il supporto a una piattaforma/acceleratore non permette il caso, rifiuto
   chiaro o limite documentato; non estendere un esito Linux a macOS/Windows.
5. Confronto della versione/revisione del binario effettivamente installato
   con quella corretta. Una build o un test unitario da soli non qualificano
   l'installazione usata dal banco Half Pocket.

La scelta di implementazione, l'eventuale forma del comando per svuotare gli
argomenti, la politica di validazione e la distribuzione spettano al team Nido.
Non e' stata preparata o applicata una patch esterna, ne un workaround launcher.

### Custodia e impatto operativo verificati ora

Il 19 ottobre 2026 alle 11:51 CEST la sola chiamata MCP di lettura `info` ha
confermato per `sdk-recovered-hub-20261019`:

```text
State: stopped
PID: 0
DiskPath: /home/jose/.nido/vms/sdk-recovered-hub-20261019.qcow2
DiskMissing: false
BackingPath: /media/jose/Data/sdk-identity-recovery-20261019/nido-custody/sdk-hub-recovery-prepared-20261019.compact.qcow2
BackingMissing: false
```

Questa risposta MCP espone la VM nell'oggetto `vm`, non in `data.vm`, e non
espone il campo `RawQemuArgs`: pertanto la lettura odierna prova lo stato e la
custodia, **non** una nuova riproduzione del campo scartato. Il risultato storico
`raw_qemu_args=null` sopra derivava dalla lettura CLI dell'analisi iniziale.

Misure di sola lettura, senza copia/compattazione dei dischi:

| Risorsa | Misura |
| --- | --- |
| Filesystem root | 314 GiB totali, circa 89 MiB disponibili, 100% nell'output `df` |
| Filesystem `/media/jose/Data` | 143 GiB totali, circa 30 GiB disponibili |
| Overlay attuale su root | 3.438.620.672 byte logici; 6.716.216 blocchi allocati da 512 byte |
| Backing su Data | 2.243.362.816 byte logici; 4.369.952 blocchi allocati da 512 byte |

La VM e il backing conservano dati utili del banco, identita, ricevute e stato
da ispezionare alla ripresa: non sono spazio sacrificabile. Non avviare il banco
con questo margine host e non dedurre sicurezza dallo spazio virtuale del guest.

**Due limiti distinti:** correggere `raw_qemu_args` consentirebbe di qualificare
la condivisione Data per i backup; non sposterebbe automaticamente l'overlay
Nido, che continua a risiedere sul filesystem root. Il riavvio del banco richiede
comunque margine host sufficiente oppure una procedura Nido supportata e
verificata per lo storage su Data. Non e' stata dimostrata una procedura
supportata di rilocazione per questa VM. Un'eventuale capacita di storage va
decisa e qualificata dal team; non e' inclusa implicitamente nella correzione
di `RawQemuArgs`.

In Half Pocket il lotto MCP e' pubblicato su `main` al commit
`6bbd8405bf5062d0ba8f120b99afe763b3f37b82`. Il checkpoint di ripresa e'
`/home/jose/halfpocket/docs/platform-sdk/GOAL.md`. Il goal resta aperto e il
banco resta fermo; le prove locali da artefatti non sono presentate come
accettazione nativa. Il team puo prendere in carico questa segnalazione senza
che l'agente esterno modifichi Nido o il suo runtime.

---

## Correzione nel checkout Nido — 19 ottobre 2026

Su mandato diretto del proprietario, il provider ora applica `RawQemuArgs`
prima di usare il salvataggio canonico. Il contratto resta a puntatori: campo
omesso invariato, lista presente sostitutiva, lista vuota rimossa. Nessun nuovo
schema, writer dello stato o percorso di compatibilita'. Le modifiche valgono
dal prossimo avvio; la configurazione funziona anche su stato di VM ferma.

La CLI espone `nido config <vm> --clear-qemu-args`, incompatibile con
`--qemu-arg`. La regola e il flag sono nel manifest, da cui derivano help e
completamenti. `--qemu-arg=` conserva una stringa vuota letterale. MCP mantiene
la distinzione fra omissione e `raw_qemu_args: []`; `null` viene rifiutato.

Verifiche eseguite su Linux con Go 1.27.2:

- `go test -p 2 ./...`: PASS.
- `go vet -p 2 ./...`: PASS.
- Regressione con overlay temporaneo che rimuove solo l'assegnazione corretta:
  i nuovi test provider, CLI e MCP falliscono sul valore persistito, come atteso.
- Test provider con stato sintetico fermo e con PID vivo del processo di test:
  impostazione, sostituzione, omissione, clear, errori senza modifica e
  conservazione degli altri campi; riapertura dello stato e generazione degli
  argomenti del prossimo avvio, senza lancio QEMU.
- Test CLI/MCP attraverso i rispettivi boundary, con provider reale in directory
  temporanee: persistenza, ordine/token, omissione, clear e richieste invalide.
  Verificati anche completamento del flag e rifiuto CLI prima della mutazione.
- Scenario `vm-mutable-config` esteso con salvataggio CLI, omissione e clear MCP;
  lo scenario con VM reali non e' stato eseguito in questa sessione.

Compilazione, dipendenze e fixture sono state confinate in una directory
temporanea in RAM. Nessuna VM esistente, disco, backing, backup o configurazione
installata e' stata modificata; nessun avvio/stop di VM o aggiornamento del
binario installato. Il file di issue preesistente e' conservato integralmente
prima di questa integrazione.

**Stato:** difetto corretto e verificato nel sorgente. Restano da qualificare
distribuzione/installazione e condivisione 9p su una VM sacrificabile con spazio
sufficiente; questi esiti non sono dedotti dai test di configurazione. La issue
resta aperta per tale accettazione operativa.

---

## Accettazione nativa Nido — 19 ottobre 2026

**Stato: chiusa dopo qualificazione del fix su Linux; distribuzione v4.5.28
attraverso la pipeline di release governata.** Questa integrazione supera il
precedente limite di sola prova sorgente, conservando le note storiche sopra.

La build corretta `v4.5.27-raw-qemu-args` e' stata installata atomicamente in
`/home/jose/.nido/bin/nido`, dopo copia verificata dei binari precedenti su Data.
La copia usata nel banco aveva lo stesso SHA-256 del binario installato.

Il banco e' un namespace di mount isolato: root in sola lettura, home Nido
temporanea montata da Data, immagine Ubuntu esistente montata in sola lettura.
Tutti gli avvii/stop/update/delete sono passati da Nido. Nessun writer manuale
dello stato e nessun avvio QEMU esterno. La VM sacrificabile aveva 2 vCPU,
1024 MiB di RAM, Ubuntu 24.04.5 e kernel 6.8.0-139-generic.

Esiti osservati:

1. CLI: configurazione `-virtfs` da VM ferma, read-back esatto, aggiornamento CPU
   con argomenti omessi, avvio e confronto con gli argomenti reali del processo.
2. Nel guest: tag `nido_qa_data`, mount 9p con `trans=virtio,version=9p2000.L`,
   lettura di un file sintetico scritto dall'host e scrittura di un file dal
   guest, verificato sul filesystem Data dell'host.
3. MCP: omissione mentre la VM era accesa senza perdita del valore; clear `[]`
   da ferma, read-back vuoto, avvio e assenza del tag nel guest.
4. MCP: nuova impostazione della lista, avvio e lettura di entrambi i file
   sintetici precedenti. CLI: clear esplicito, stop/start e nuova assenza del tag.
5. File condivisi preservati dopo i clear; VM sacrificabile rimossa tramite
   Nido; flotta e template del banco vuoti al termine.

Validazione di deploy locale: suite Go completa e lint PASS; blackbox completo
48 PASS, 0 FAIL, 3 SKIP previsti (GUI, self-update e rimozione di immagine ancora
referenziata). Anche le VM/template del validator sono state eliminate nel banco.

Le 7 VM preesistenti sono rimaste ferme e intatte: nomi, dimensioni, inode e
mtime dei file VM/runtime confrontati con la baseline; hash JSON di stato
invariati. Nessuna copia, rilocazione o avvio delle loro immagini/backing.
La directory condivisa conteneva soltanto sentinelle sintetiche della prova.
Gli esiti sono qualificazione Linux/QEMU/KVM, non una prova macOS/Windows.
