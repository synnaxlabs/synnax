# AI voiceover for auto-generated docs videos: research report

Scope: batch TTS for 60–120 s English docs/tutorial videos, calm professional
narration, published as marketing/docs content. Latency irrelevant. Findings as of
August 2026.

---

## 1. Provider comparison

| Provider / model | Quality tier (narration) | Price (~per 1k chars / per min*) | Word timestamps | Commercial license | Pronunciation control | Voice consistency / pinning |
|---|---|---|---|---|---|---|
| **ElevenLabs** `eleven_multilingual_v2`, `eleven_v3` | **A** (v3 leads expressive narration; v2 is the stable workhorse) | $0.10/1k chars (v2, v3); $0.05/1k (Flash/Turbo). ~$0.09–0.10/min at ~950 chars/min | **Yes, native**: `/v1/text-to-speech/{voice_id}/with-timestamps` returns character-level start/end times (roll up to words); v3 has word-level timing; separate Forced Alignment API too | Commercial use on any **paid** plan (free tier = attribution, non-commercial) | Pronunciation dictionaries (PLS): **alias rules work on all models**; IPA/CMU **phoneme tags only on Flash v2/Turbo v2 and v3, NOT multilingual v2** | Strong: use your own PVC or a purchased/designed voice; library voices can be pulled by their owners, so pin a voice you own. Model IDs are versioned (`eleven_multilingual_v2` stable for 2+ yrs); `eleven_monolingual_v1` deprecation shows old models do get removed eventually |
| **OpenAI** `gpt-4o-mini-tts` | **B+** (good, conversational-grade; steerable via `instructions` prompt) | ~**$0.015/min** ($12/1M audio tokens + $0.60/1M text tokens) — cheapest hosted quality option | **No.** No timestamps in TTS API. Must force-align afterward (e.g. WhisperX) or transcribe with `whisper-1`/`gpt-4o-transcribe` word timestamps | Outputs owned by you, commercial use fine; usage policy requires disclosing voice is AI-generated | **None structured**: no SSML, no lexicons. Only prose `instructions` + respelling in text. `speed` param reported **ignored** on gpt-4o-mini-tts ([community thread](https://community.openai.com/t/new-tts-model-gpt-4o-mini-tts-ignoring-speed-parameter/1154883)) | Weak: 13 fixed voices, no cloning, generations are non-deterministic in pacing/tone; fine voice identity stability, but delivery varies run to run |
| **Cartesia** Sonic-3 / Sonic-3.6 | **A** (Sonic-3.6 currently leads both Artificial Analysis speech arenas) | 1 credit/char; effective **$5–$37 per 1M chars** by plan (~$0.005–0.037/1k) — up to ~4x cheaper than ElevenLabs. Pro cloning: one-time training fee + 1.5 credits/char | **Yes, native**: `add_timestamps` returns word-level timings (docs note start-of-word timings need testing; end-of-word more reliable) | Commercial use on paid API plans | Good: inline MFA-style IPA `<<phonemes>>` custom pronunciations; SSML-ish tags for speed/volume/emotion | Good: cloned/owned voices with stable IDs; model line moves fast (3 → 3.5 → 3.6 in months) so pin the model version string |
| **Google Cloud** Chirp 3 HD / Gemini-TTS | **A−/A** (Gemini TTS ranks near top of arenas; Chirp 3 HD solid narration) | Chirp 3 HD: **$30/1M chars** (1M free/mo). Gemini 2.5 Flash TTS: $10/1M audio tokens ≈ ~$0.024/min; Pro 2x | **No word timestamps from TTS.** (STT side has them.) Chirp 3 HD supports timepointing only via SSML marks on some voices — not reliable; plan on forced alignment | Standard GCP terms, commercial fine | Chirp 3 HD: SSML subset incl. `<phoneme>` (IPA/X-SAMPA), `<sub>`, custom pronunciations — but users report custom pronunciations **not working** on Chirp 3 HD ([forum](https://discuss.ai.google.dev/t/custom-pronunciations-not-working-with-chirp3-hd-voices/106259)). Gemini-TTS: prompt-steered, no SSML | Fixed named voices, very stable over time (GCP deprecation policy); Instant Custom Voice available |
| **Amazon Polly** generative | **B** (clearly behind ElevenLabs/Cartesia for naturalness) | Generative: **$30/1M chars**; Neural $16; Long-form $100 | **Yes, native**: Speech Marks give word + sentence (+ viseme) timestamps, billed at same per-char rate as a second request | AWS terms, commercial fine | Best-in-class classic controls: full SSML, uploadable PLS **lexicons** (IPA/X-SAMPA), `<sub>`, `<prosody rate>` | Excellent: voices are long-lived, versioned engine names; AWS rarely removes voices |
| **MiniMax** speech-2.8-hd / speech-02-hd | **A−** (arena ELO just under ElevenLabs v3) | **$100/1M chars** ($0.10/1k) direct; cheaper via resellers (fal, WaveSpeed) | Partial: `subtitle_enable` returns sentence-level timing on async T2A; no word-level — force-align for word sync | Commercial on paid API; China-based vendor — check data/procurement comfort | `pronunciation_dict` (respelling mappings, e.g. `"API/A P I"`), speed/pitch/vol params | Named voice IDs (`English_expressive_narrator`), cloning $1.50/voice; model line churns quickly |
| **PlayHT** (Play 3.0 mini / PlayDialog) | **B** (mindshare and arena standing have slipped; pivoted to voice agents) | Subscription-oriented; opaque API per-char pricing (Creator/Unlimited plans; "Unlimited" ~$299/mo tier historically) | Yes on some endpoints (viseme/word events on streaming), poorly documented | Commercial on paid plans | Basic; limited SSML | Cloned voices persist; company direction uncertain — weakest bet of the hosted A/B tier |
| **Kokoro-82M** (open) | **B** for docs narration (remarkably good for 82M; flatter prosody, occasional artifacts; small voice set) | **~$0 marginal** (self-host; runs realtime on CPU) — Apache-2.0 | No native timestamps (some wrappers expose phoneme durations since it's duration-model based; practical route is forced alignment) | **Apache-2.0 — full commercial use, safe for published marketing content** | eSpeak-based G2P: you can inject phonemes/respellings at the text layer; full control since you own the stack | **Perfect pinning**: weights are frozen files; the same input yields the same voice forever |
| **F5-TTS** (open) | A− cloning quality | free | No | ❌ **Weights CC-BY-NC-4.0 (Emilia dataset) — NOT usable for published commercial content** (code is MIT, irrelevant) | — | — |
| **XTTS v2 / Coqui** (open) | B | free | No | ❌ **CPML non-commercial; Coqui shut down 2024, no one can sell you a license** | — | — |

\*Assume docs narration ≈ 950–1,050 chars per audio minute at ~150 wpm. At a scale of
100 videos/mo x 100 s ≈ 170 min/mo ≈ 175k chars/mo: ElevenLabs v2 ≈ $17.50/mo of
credits (fits the $22 Creator plan), Cartesia ≈ $1–6, OpenAI ≈ $2.60, Polly generative
≈ $5.25. Cost is a non-issue at this volume; choose on quality + timestamps +
pronunciation control.

Sources: [ElevenLabs pricing breakdown](https://flexprice.io/blog/elevenlabs-pricing-breakdown), [ElevenLabs API pricing](https://unifically.com/blogs/elevenlabs), [gpt-4o-mini-tts specs/pricing](https://gate.ai/blog/gpt-4o-mini-tts-openai-specs-pricing-api-use-cases), [OpenAI TTS pricing compared](https://texttolab.com/blog/openai-tts-pricing), [Cartesia pricing](https://texttolab.com/blog/cartesia-pricing), [Cartesia Sonic-3.6 arena lead](https://www.marktechpost.com/2026/08/18/cartesia-ships-sonic-3-6-a-streaming-tts-model-that-now-leads-both-artificial-analysis-speech-arenas/), [Google Cloud TTS pricing](https://texttolab.com/blog/google-cloud-tts-pricing), [Polly pricing](https://texttolab.com/blog/amazon-polly-pricing) + [aws.amazon.com/polly/pricing](https://aws.amazon.com/polly/pricing/), [MiniMax pricing](https://minimax-ai.chat/pricing/), [TTS model benchmark comparison](https://www.marktechpost.com/2026/05/30/best-text-to-speech-tts-models-in-2026-a-benchmark-based-comparison/), [Kokoro-82M (Apache-2.0)](https://huggingface.co/hexgrad/Kokoro-82M), [XTTS commercial-use status](https://localaimaster.com/blog/xtts-coqui-commercial-license), [Kokoro vs XTTS licensing](https://localaimaster.com/blog/kokoro-vs-xtts-vs-chatterbox).

---

## 2. Word timestamps and forced alignment

**Native from the TTS API (zero extra work, perfectly ground-truth):**
- **ElevenLabs**: `POST /v1/text-to-speech/{voice_id}/with-timestamps` (and
  `/stream/with-timestamps`) return `alignment.characters` +
  `character_start_times_seconds` / `character_end_times_seconds`, plus
  `normalized_alignment` (post-normalization text, the one to use). Roll characters
  into words by splitting on spaces. Works with `eleven_multilingual_v2`; `eleven_v3`
  now also ships word-level timing. Docs:
  [convert-with-timestamps](https://elevenlabs.io/docs/api-reference/text-to-speech/convert-with-timestamps),
  [announcement](https://elevenlabs.io/blog/new-text-to-speech-endpoints-with-timestamps).
  ElevenLabs also sells a standalone
  [Forced Alignment API](https://elevenlabs.io/docs/overview/capabilities/forced-alignment)
  (audio + transcript → word timings) for mixed-in human VO.
- **Cartesia**: `add_timestamps: true` on the TTS request returns word-level
  timestamps. Their docs recommend trusting end-of-word timestamps more than
  start-of-word ([changelog](https://docs.cartesia.ai/changelog/2026)).
- **Amazon Polly**: Speech Marks request (`SpeechMarkTypes: ["word","sentence"]`) — a
  second call with the same text, billed per character again. Rock solid, exact.
- **MiniMax**: sentence-level subtitles only (`subtitle_enable` on async T2A) — not
  word-level.

**Not native:** OpenAI TTS (nothing), Google Chirp 3 HD / Gemini-TTS (nothing usable
for word timing).

**Forced-alignment fallback (when TTS gives no timings):** since we have the exact
narration text, alignment is the easy case — no ASR errors to fight.
- **Montreal Forced Aligner (MFA)** — most accurate: GMM-HMM at 10 ms resolution; a
  2026 evaluation shows it beats WhisperX and MMS at all time resolutions
  ([arXiv 2606.18466](https://arxiv.org/html/2606.18466v1),
  [Interspeech 2024 comparison](https://www.isca-archive.org/interspeech_2024/rousso24_interspeech.pdf)).
  Cost: heavyweight install (conda, acoustic model + dictionary downloads), OOV
  handling needed for "Synnax"/"Cesium" (G2P or custom dict entries). Best when you
  need phoneme-grade precision — overkill here.
- **WhisperX** — easiest pipeline fit: pip install, wav2vec2 phoneme alignment on top
  of Whisper; word timestamps good to roughly ±50–100 ms, with known drift cases vs
  MFA ([whisperX issue #1247](https://github.com/m-bain/whisperX/issues/1247)). Fine
  for caption timing and hold-stretching (~100 ms accuracy suffices to sync a sentence
  boundary to a click).
- **aeneas** — DTW/espeak-based, designed for sentence/paragraph-level audiobook sync;
  word-level accuracy is poor. Effectively unmaintained.

Practical ruling: prefer a TTS that returns timestamps natively; keep WhisperX as the
generic fallback; skip MFA unless caption QA shows drift.

---

## 3. Pacing/fit: TTS-first vs video-first

**Prior art converges on TTS-first (audio is the master clock):**
- **Synthesia**: scene duration is derived from the script audio — the avatar speaks,
  and the scene runs as long as the narration
  ([script docs](https://docs.synthesia.io/docs/script); uploaded audio caps at 5
  min/scene). Video accommodates audio, never the reverse.
- **Guidde**: records each action as a discrete step, generates per-step narration,
  and each step's screen time is the narration length plus padding
  ([guidde.com](https://www.guidde.com/)). Steps are the unit of sync.
- **Clueso / Trupeer**: same shape — script is segmented per captured action;
  voiceover is generated per segment; zoom/highlight effects are keyed to the click
  events and the segment is held for the narration duration; "edit the script and the
  video re-syncs" is the core UX
  ([Clueso screen recorder](https://www.clueso.io/features/screen-recorder),
  [Clueso](https://www.clueso.io/)).

Nobody in this space does video-first (fixed gaps, TTS speed-warped to fit) as the
primary mechanism:
- **Speed-control fidelity is poor across modern neural TTS.** OpenAI's `speed` param
  is reportedly ignored on gpt-4o-mini-tts; ElevenLabs exposes only a coarse `speed`
  voice setting (0.7–1.2, quality degrades at the edges); Cartesia's speed control is
  a small range; only Polly's `<prosody rate>` is truly faithful, and Polly is the
  weakest voice. Time-stretching rendered audio >±8–10% (even with ffmpeg
  `atempo`/rubberband) is audible on speech.
- We control the video timeline exactly (scripted browser session), so stretching a
  hold frame is free and invisible; warping speech is neither.

**Recommended hybrid (what the polished tools effectively do):**
1. Segment the script per action/step (we already have the action timeline).
2. Render narration **per segment** (one TTS call per step, or one call with per-word
   timestamps and split on sentence boundaries). Per-segment calls give natural
   re-render granularity when one sentence changes.
3. For each step: `screen_time = max(min_action_time, narration_duration + lead_in +
   tail_pad)`. Narration for step N starts ~300–500 ms before the visual action
   completes (voice slightly leads or lands with the click, never lags).
4. Only if a segment's narration is grossly long relative to a hard visual constraint,
   apply mild TTS speed (≤1.1x) or rewrite the sentence — script rewriting is the real
   length knob, and it's automatable (LLM: "shorten to ≤ N words").
5. Concatenate audio with fixed inter-step silences; the video renderer consumes the
   resulting per-step durations.

---

## 4. Captions

- **Produce both**: a **WebVTT/SRT sidecar** for the docs-site `<video>`/player and
  YouTube (accessibility, WCAG/ADA, SEO, user toggle), and optionally a **burned-in**
  variant only for social cuts. Burned-in as the *only* captions fails accessibility
  guidance
  ([BOIA on burned-in captions](https://www.boia.org/blog/how-do-burned-in-captions-affect-accessibility),
  [Section 508](https://www.section508.gov/create/captions-transcripts/)). Since we
  know the script, the VTT is free — cue it from the same word timestamps.
- Styling/segmentation
  ([3Play best practices](https://www.3playmedia.com/blog/closed-caption-styling-formatting-best-practices-you-need-to-know/),
  [subtitle standards](https://videotap.com/blog/subtitle-formatting-best-practices-and-standards)):
  ≤42 chars/line, max 2 lines, break at clause boundaries (never orphan an
  article/preposition at line end), 1–6 s per cue, white sans-serif on
  semi-transparent dark band, bottom-centered but lifted above player UI. For docs
  videos specifically, keep cues sentence-aligned with the narration segments so a
  caption never straddles two on-screen actions.

---

## 5. Pronunciation of "Synnax", "Cesium", "Pluto", "LabJack", "OPC UA"

Per provider:
- **ElevenLabs**: pronunciation dictionary (PLS) attached per-request via
  `pronunciation_dictionary_locators`. **Alias rules work on every model** — use them
  for `OPC UA → "O P C U A"`, `LabJack → "Lab Jack"`, `Synnax → "sin-ax"` respelling.
  **IPA phoneme rules do NOT work on `eleven_multilingual_v2`** (silently skipped) —
  they need Flash v2/Turbo v2 or v3
  ([docs](https://elevenlabs.io/docs/eleven-api/guides/how-to/text-to-speech/pronunciation-dictionaries)).
  Practical: aliases (respellings) are enough for this term list; "Cesium" and
  "Pluto" are dictionary words that all engines say correctly.
- **Cartesia**: inline MFA-IPA `<<s|ɪ|n|æ|k|s>>` custom pronunciations, plus plain
  respelling
  ([custom pronunciations docs](https://docs.cartesia.ai/build-with-cartesia/sonic-3/custom-pronunciations)).
- **OpenAI**: no lexicon, no SSML — respell directly in the input text and keep
  display text separate from spoken text in the pipeline.
- **Google Chirp 3 HD**: documented `<phoneme>`/custom pronunciations (IPA/X-SAMPA),
  but real-world reports of it not working; Gemini-TTS is prompt-only.
- **Polly**: uploadable lexicons (PLS, IPA) — the gold standard mechanism, on the
  weakest voice.
- **MiniMax**: `pronunciation_dict` respelling mappings.

Architecture note regardless of provider: keep a **spoken-form dictionary** in the
video pipeline (`display: "OPC UA"` → `spoken: "O P C U A"`), applied to the TTS input
only. That makes the pipeline provider-portable, and captions still show the correct
written form. Timestamp mapping survives because ElevenLabs returns
`normalized_alignment` and we control the substitution offsets.

---

## Recommendation

**Primary: ElevenLabs `eleven_multilingual_v2` (moving to `eleven_v3` after a
bake-off), Creator plan ($22/mo covers the volume), one Professional Voice Clone or
one purchased library voice pinned by voice_id.**
- Only top-tier provider with native character/word timestamps AND mature
  pronunciation dictionaries AND calm-narration voices proven at scale in exactly this
  product category (Clueso, Guidde et al. are widely reported ElevenLabs customers).
- Pin `model_id` + `voice_id` + `voice_settings` (stability high ~0.6–0.75, style low)
  in config; `eleven_multilingual_v2` has been stable for over two years, so
  month-to-month consistency across the video library is realistic. Avoid free-tier
  (non-commercial) and avoid community library voices you don't own (owner can
  withdraw them) — clone or buy.

**Fallback A (hosted, cheap, arena-leading): Cartesia Sonic-3.x** — native word
timestamps, inline IPA, ~4x cheaper. Main risks: fast model churn (pin version
strings) and a younger long-form-narration track record.

**Fallback B (zero-cost, zero-dependency, perfectly reproducible): self-hosted
Kokoro-82M (Apache-2.0) + WhisperX alignment.** Quality is a clear step down but
acceptable for internal/low-stakes videos, and it is the only open option that is
licensing-safe — **do not use F5-TTS (CC-BY-NC weights) or XTTS (CPML, unlicensable
since Coqui's shutdown) for published marketing content.**

Avoid: OpenAI TTS as primary (no timestamps, no lexicon, broken speed param,
non-deterministic delivery — cheapness doesn't matter at this volume); PlayHT (opaque
pricing, drifting focus); Polly (mechanism-rich, voice-poor).

**Recommended sync architecture (TTS-first, per-step):**
1. Script generation produces per-action narration segments with spoken-form
   dictionary applied.
2. One ElevenLabs `with-timestamps` call per segment → WAV + word timings (from
   character alignment).
3. Timeline solver sets each step's hold: `max(min_visual_time, audio_len + pads)`;
   narration onset leads the click by ~300 ms.
4. Renderer stretches holds; audio track is concatenated segments + sized silences.
5. Captions: cue file generated from the same word timings (≤42 chars/line, 2 lines,
   sentence-aligned cues) → ship WebVTT with the docs player; burn in only for social
   exports.
6. Regeneration: content-hash each segment (text + voice_id + model_id + settings) and
   re-render only changed segments — keeps cost near zero and voice consistent across
   hundreds of videos.
