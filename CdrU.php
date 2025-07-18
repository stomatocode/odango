<?php

class CdrU extends AppModel
{
    public $useDbConfig = 'cdr';

    public $nsPrimaryKey = 'cdr_id';

    public $schemaApiToDb = array(
        'uid' => 'uid',
        'cdr_id' => 'cdr_id',
        'type' => 'type',
        'time_start' => 'time_start',
        'time_answer' => 'time_answer',
        'time_release' => 'time_release',
        'duration' => 'duration',
        'orig_to_uri' => 'orig_to_uri',
        'number' => 'number',
        'name' => 'name',
        'onnet' => 'onnet',
        'hide' => 'hide',
        'tag' => 'tag',
        'transcription_job_id' => 'transcription_job_id',
        'sentiment_positive_percent' => 'sentiment_positive_percent',
        'sentiment_neutral_percent' => 'sentiment_neutral_percent',
        'sentiment_negative_percent' => 'sentiment_negative_percent',
        'ending_sentiment' => 'ending_sentiment',
        'top_topics' => 'top_topics',
    );

    public $preferedFieldOrder = array(
        'id',
        'domain',
        'uid',
    );
}