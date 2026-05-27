{-# LANGUAGE OverloadedStrings #-}
{-# OPTIONS_GHC -Wall -Werror #-}
{-# LANGUAGE InstanceSigs #-}
module TextMate where

import Data.Aeson
import Data.ByteString.Lazy (ByteString)
import Data.Maybe (catMaybes)
import qualified Data.Map.Strict as Map
-- structure of the textmate json

type ScopeName = String
type Regex = String
type CaptureKey = String
type Capture = (CaptureKey, ScopeName)
type Captures = [Capture]
type RepoKey = String
type Named = (RepoKey, TmPattern)

-- datatype representing the regexes for a single object
data TmPattern
  = TmMatch   (Maybe ScopeName) Regex Captures
  | TmRegion  (Maybe ScopeName) Regex Regex Captures Captures [TmPattern]
  --           name              begin end   beginCaps  endCaps   inner
  | TmInclude RepoKey

capturesJSON :: Captures -> Value
capturesJSON caps = toJSON $ Map.fromList [(k, object ["name" .= v]) | (k, v) <- caps]

instance ToJSON TmPattern where
  toJSON :: TmPattern -> Value
  toJSON (TmMatch name match captures) = object $ catMaybes
    [ ("name" .=) <$> name
    , Just $ "match" .= match
    , ("captures" .=) . capturesJSON <$> omitEmpty captures
    ]
  toJSON (TmRegion name begin end bcaps ecaps inner) = object $ catMaybes
    [ ("name" .=) <$> name
    , Just $ "begin" .= begin
    , Just $ "end" .= end
    , ("beginCaptures" .=) . capturesJSON <$> omitEmpty bcaps
    , ("endCaptures" .=) . capturesJSON <$> omitEmpty ecaps
    , ("patterns" .=) <$> omitEmpty inner
    ]
  toJSON (TmInclude rk) = object ["include" .= ('#' : rk)]

-- datatype representing the whole syntax file
data TmSyntax
  = TmSyntax String [String] [String] [TmPattern] [Named]
  --          scope  types    exts    top-patterns repository

instance ToJSON TmSyntax where
  toJSON (TmSyntax scopeName fileTypes fileExts patterns repo) = object $ catMaybes
    [ Just $ "$schema"       .= ("https://raw.githubusercontent.com/martinring/tmlanguage/master/tmlanguage.json" :: String)
    , Just $ "scopeName"     .= scopeName
    , ("fileTypes" .=)       <$> omitEmpty fileTypes
    , ("fileExtensions" .=)  <$> omitEmpty fileExts
    , ("patterns" .=)        <$> omitEmpty patterns
    , ("repository" .=) . Map.fromList <$> omitEmpty repo
    ]


omitEmpty :: [a] -> Maybe [a]
omitEmpty [] = Nothing
omitEmpty xs@(_:_) = Just xs

syntaxToJson :: TmSyntax -> ByteString
syntaxToJson = encode